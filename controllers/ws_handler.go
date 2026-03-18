package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"webmajiang/models"
	"webmajiang/models/pb"
	"webmajiang/service"
	"webmajiang/utils"

	"strconv"

	"github.com/maoxiaoyue/hypgo/pkg/websocket"
	"google.golang.org/protobuf/proto"
)

func keepOnline(playerIDStr string) {
	if playerIDStr == "" {
		return
	}
	id, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		return
	}
	ctx := context.Background()
	user, err := models.GetUserByID(ctx, id)
	if err == nil {
		models.KeepUserOnline(ctx, user.ID, user.Username)
	}
}

// globalHub 全域 Hub 參考，供 game_loop 廣播用
var globalHub *websocket.Hub

// BroadcastBotSpeech 廣播 Bot 說話訊息
func BroadcastBotSpeech(playerID int, content string) {
	if globalHub == nil || content == "" {
		return
	}
	payload := map[string]interface{}{
		"player_id": playerID,
		"content":   content,
	}
	jsonBytes, _ := json.Marshal(payload)
	speech := &pb.PlayerActionRes{
		Success: true,
		Message: string(jsonBytes),
	}
	sendProtoBroadcast(globalHub, "bot_speech", speech)
}

// HandleWebSocketMessage acts as the main router for incoming WebSocket messages
// v0.7.1: hypgo ProtobufCodec 已處理外層 WsMessage 信封，
// msg.Type = action 名稱, msg.Data = 內層 protobuf bytes
func HandleWebSocketMessage(client *websocket.Client, msg *websocket.Message) {
	ctx := context.Background()

	action := msg.Type
	data := []byte(msg.Data) // json.RawMessage → []byte (即內層 protobuf payload)

	switch action {
	// === 玩家加入房間 ===
	case "join_room":
		handleJoinRoom(ctx, client, action, data)

	// === 決定座位 (擲骰子) ===
	case "roll_positions":
		handleRollPositions(ctx, client, action, data)

	// === 決定莊家 (擲骰子) ===
	case "roll_dealer":
		handleRollDealer(ctx, client, action, data)

	// === 觸發發牌 ===
	case "deal_tiles":
		handleDealTiles(ctx, client, action, data)

	// === 手牌排序 ===
	case "sort_hand":
		handleSortHand(ctx, client, action, data)

	// === 玩家摸牌 ===
	case "draw_tile":
		handleDrawTile(ctx, client, action, data)

	// === 玩家出牌 ===
	case "discard_tile":
		handleDiscardTile(ctx, client, action, data)

	// === 暗槓 ===
	case "concealed_kong":
		handleConcealedKong(ctx, client, action, data)

	// === 加槓 ===
	case "add_kong":
		handleAddKong(ctx, client, action, data)

	// === 自摸 ===
	case "self_draw_hu":
		handleSelfDrawHu(ctx, client, action, data)

	// === 玩家宣告 (吃/碰/槓/胡/放棄) ===
	case "player_action":
		handlePlayerAction(ctx, client, action, data)

	// === 進入下一局 ===
	case "next_round":
		handleNextRound(ctx, client, action, data)

	// === 查詢目前遊戲狀態 ===
	case "get_state":
		handleGetState(ctx, client, action, data)

	// === 查詢各家手牌 ===
	case "get_hands":
		handleGetHands(ctx, client, action, data)

	// === 查詢牌堆剩餘數量 ===
	case "get_deck_count":
		handleGetDeckCount(ctx, client, action, data)

	default:
		utils.Info("Unhandled websocket action type: %s", action)
	}
}

// =====================================
// 各路由 Handler
// =====================================

func handleJoinRoom(ctx context.Context, client *websocket.Client, action string, data []byte) {
	// 保存全域 Hub 供 game_loop 廣播用
	if client.Hub != nil {
		globalHub = client.Hub
	}

	var joinReq pb.JoinRoomReq
	if err := proto.Unmarshal(data, &joinReq); err != nil {
		sendWSError(client, action, "invalid JoinRoomReq data")
		return
	}

	gameID := joinReq.RoomId
	playerID := 1
	if joinReq.PlayerId != "" {
		fmt.Sscanf(joinReq.PlayerId, "%d", &playerID)
	}

	utils.Info("[WS] handleJoinRoom: gameID=%q, playerID=%d, rawRoomId=%q, rawPlayerId=%q",
		gameID, playerID, joinReq.RoomId, joinReq.PlayerId)

	// 儲存 gameID 和 playerID 到 client metadata
	client.SetMetadata("gameID", gameID)
	client.SetMetadata("playerID", playerID)

	go keepOnline(joinReq.PlayerId)

	// 回覆加入成功
	res := &pb.JoinRoomRes{
		Success: true,
		Message: "加入成功",
	}
	sendProtoResponse(client, action+"_res", res)

	// 載入遊戲狀態
	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		utils.Error("[WS] handleJoinRoom: LoadGameState 失敗: gameID=%q, err=%v", gameID, err)
		sendWSError(client, action, "載入遊戲狀態失敗: "+err.Error())
		return
	}

	utils.Info("[WS] handleJoinRoom: 載入成功, stage=%q, players=%d", state.Stage, len(state.Players))

	// 自動啟動遊戲流程：WAITING_PLAYERS → 擲骰 → 決定莊家 → 發牌
	if state.Stage == models.StageWaitingPlayers {
		utils.Info("[WS] 自動啟動遊戲流程: %s", gameID)

		// 1. 擲骰決定座位
		state, err = RollPositions(ctx, gameID)
		if err != nil {
			utils.Error("[WS] RollPositions 失敗: %v", err)
			syncData := buildSyncStateData(gameID, state)
			sendProtoResponse(client, "sync_state", syncData)
			return
		}

		// 2. 擲骰決定莊家
		state, err = RollDealer(ctx, gameID)
		if err != nil {
			utils.Error("[WS] RollDealer 失敗: %v", err)
			syncData := buildSyncStateData(gameID, state)
			sendProtoResponse(client, "sync_state", syncData)
			return
		}

		// 3. 發牌
		state, err = DealTilesAction(ctx, gameID)
		if err != nil {
			utils.Error("[WS] DealTilesAction 失敗: %v", err)
			syncData := buildSyncStateData(gameID, state)
			sendProtoResponse(client, "sync_state", syncData)
			return
		}

		// 發送玩家自己的手牌
		hand, err := GetPlayerHand(ctx, gameID, playerID)
		if err == nil {
			tileIds := make([]int32, len(hand))
			for i, t := range hand {
				tileIds[i] = int32(t.ID)
			}
			sendProtoResponse(client, "deal_tiles", &pb.DealTilesData{Tiles: tileIds})
		}

		// 廣播最新狀態
		syncData := buildSyncStateData(gameID, state)
		sendProtoBroadcast(client.Hub, "sync_state", syncData)

		// 如果莊家是 AI，等待前端發牌動畫後再自動出牌
		dealer, ok := state.Players[state.CurrentPlayerID]
		if ok && dealer.IsBot {
			go func() {
				time.Sleep(6 * time.Second) // 等待前端發牌動畫完成
				if err := ProcessAITurn(context.Background(), gameID, dealer); err != nil {
					utils.Error("[WS] AI 莊家自動出牌失敗: %v", err)
				}
				if newState, err := LoadGameState(context.Background(), gameID); err == nil {
					syncData := buildSyncStateData(gameID, newState)
					sendProtoBroadcast(client.Hub, "sync_state", syncData)
				}
			}()
		}
	} else {
		// 遊戲已在進行中，只同步當前狀態
		syncData := buildSyncStateData(gameID, state)
		sendProtoResponse(client, "sync_state", syncData)
	}
}

func handleRollPositions(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid request data")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}

	go keepOnline(req.PlayerId)

	state, err := RollPositions(ctx, gameID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	// 回覆擲骰結果
	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: fmt.Sprintf("骰子結果: %d + %d = %d", state.Dice.Die1, state.Dice.Die2, state.Dice.Total),
	})

	// 廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)
}

func handleRollDealer(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid request data")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}

	go keepOnline(req.PlayerId)

	state, err := RollDealer(ctx, gameID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: fmt.Sprintf("莊家決定: 玩家 %d (骰子: %d)", state.DealerPlayerID, state.Dice.Total),
	})

	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)
}

func handleDealTiles(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid JoinRoomReq data for deal_tiles")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}
	requesterID := 1
	if req.PlayerId != "" {
		// 嘗試解析 player_id
		fmt.Sscanf(req.PlayerId, "%d", &requesterID)
		go keepOnline(req.PlayerId)
	}

	// 執行發牌
	state, err := DealTilesAction(ctx, gameID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	// 回傳請求者自己的手牌 tile ID list
	hand, err := GetPlayerHand(ctx, gameID, requesterID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	tileIds := make([]int32, len(hand))
	for i, t := range hand {
		tileIds[i] = int32(t.ID)
	}

	dealRes := &pb.DealTilesData{
		Tiles: tileIds,
	}
	sendProtoResponse(client, "deal_tiles_res", dealRes)

	// 廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)

	// 如果莊家是 AI，自動觸發莊家出牌
	dealer, ok := state.Players[state.CurrentPlayerID]
	if ok && dealer.IsBot {
		go func() {
			if err := ProcessAITurn(context.Background(), gameID, dealer); err != nil {
				utils.Error("[WS] AI 莊家自動出牌失敗: %v", err)
			}
			// 出牌後廣播最新狀態
			if newState, err := LoadGameState(context.Background(), gameID); err == nil {
				syncData := buildSyncStateData(gameID, newState)
				sendProtoBroadcast(client.Hub, "sync_state", syncData)
			}
		}()
	}
}

func handleSortHand(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid sort_hand request")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}
	playerID := 1
	if req.PlayerId != "" {
		fmt.Sscanf(req.PlayerId, "%d", &playerID)
		go keepOnline(req.PlayerId)
	}

	if err := SortPlayerHand(ctx, gameID, playerID); err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	sortedHand, err := GetPlayerHand(ctx, gameID, playerID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	tileIds := make([]int32, len(sortedHand))
	for i, t := range sortedHand {
		tileIds[i] = int32(t.ID)
	}

	sendProtoResponse(client, "sort_hand_res", &pb.DealTilesData{
		Tiles: tileIds,
	})
}

func handleDrawTile(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid draw_tile request")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}
	playerID := 1
	if req.PlayerId != "" {
		fmt.Sscanf(req.PlayerId, "%d", &playerID)
		go keepOnline(req.PlayerId)
	}

	state, drawnTile, err := DrawTileAction(ctx, gameID, playerID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	// 荒莊流局
	if drawnTile == nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: true,
			Message: "荒莊流局，牌堆已空",
		})
		syncData := buildSyncStateData(gameID, state)
		sendProtoBroadcast(client.Hub, "sync_state", syncData)
		return
	}

	// 回傳摸到的牌
	sendProtoResponse(client, action+"_res", &pb.DealTilesData{
		Tiles: []int32{int32(drawnTile.ID)},
	})

	// 廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)
}

func handleConcealedKong(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var actionReq pb.PlayerActionData
	if err := proto.Unmarshal(data, &actionReq); err != nil {
		sendWSError(client, action, "invalid concealed_kong data")
		return
	}

	gameID := getClientGameID(client)
	playerID := getClientPlayerID(client)

	// tileId 用來識別要槓哪種牌 — 從 0-based tile ID 查出花色和數值
	tileID := int(actionReq.TileId)
	tile := tileFromID(tileID)
	if tile == nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: fmt.Sprintf("無效的牌 ID: %d", tileID),
		})
		return
	}

	state, err := ConcealedKongAction(ctx, gameID, playerID, tile.Type, tile.Value)
	if err != nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: "暗槓成功",
	})

	// 廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)
}

func handleSelfDrawHu(ctx context.Context, client *websocket.Client, action string, data []byte) {
	gameID := getClientGameID(client)
	playerID := getClientPlayerID(client)

	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: "載入遊戲狀態失敗: " + err.Error(),
		})
		return
	}

	if state.Stage != models.StagePlayerDiscard {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: fmt.Sprintf("自摸不允許在此階段: %s", state.Stage),
		})
		return
	}

	if state.CurrentPlayerID != playerID {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: "不是你的回合",
		})
		return
	}

	// 取得手牌驗證是否真的可以胡
	hand, err := GetPlayerHand(ctx, gameID, playerID)
	if err != nil || !models.CanHu(hand) {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: "手牌不符合胡牌條件",
		})
		return
	}

	// 喊「自摸」
	BroadcastBotSpeech(playerID, "自摸！")

	// 廣播當前狀態讓大家看到
	syncData := buildSyncStateData(gameID, state)
	if client.Hub != nil {
		sendProtoBroadcast(client.Hub, "sync_state", syncData)
	}
	time.Sleep(2 * time.Second)

	// 設定 ROUND_OVER
	state.Stage = models.StageRoundOver
	state.CurrentPlayerID = playerID
	state.WinnerIDs = []int{playerID}
	if err := SaveGameState(ctx, state); err != nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: "儲存狀態失敗: " + err.Error(),
		})
		return
	}

	// 執行計分（呼叫 ResolveActions 中相同的計分邏輯）
	rdb := service.RedisClient
	var melds []models.Meld
	meldsKey := PlayerMeldsKey(gameID, playerID)
	meldJSONs, _ := rdb.LRange(ctx, meldsKey, 0, -1).Result()
	for _, mj := range meldJSONs {
		var m models.Meld
		if json.Unmarshal([]byte(mj), &m) == nil {
			melds = append(melds, m)
		}
	}
	var flowers []models.Tile
	flowersKey := PlayerFlowersKey(gameID, playerID)
	flowerJSONs, _ := rdb.LRange(ctx, flowersKey, 0, -1).Result()
	for _, fj := range flowerJSONs {
		var t models.Tile
		if json.Unmarshal([]byte(fj), &t) == nil {
			flowers = append(flowers, t)
		}
	}

	scoreCtx := models.ScoringContext{
		ClosedHand:  hand,
		Melds:       melds,
		IsSelfDrawn: true,
		IsDealer:    state.DealerPlayerID == playerID,
		Flowers:     flowers,
	}
	scoreResult := models.CalculateScore(scoreCtx)
	if state.ScoreResults == nil {
		state.ScoreResults = make(map[int]models.ScoreResult)
	}
	state.ScoreResults[playerID] = scoreResult
	if err := SaveGameState(ctx, state); err != nil {
		utils.Error("[SelfDrawHu] 計分儲存失敗: %v", err)
	}

	utils.Info("[SelfDrawHu] 玩家 %d 自摸！TotalTai: %d", playerID, scoreResult.TotalTai)

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: "自摸成功",
	})

	// 廣播結算狀態
	finalSyncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", finalSyncData)
}

func handleAddKong(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var actionReq pb.PlayerActionData
	if err := proto.Unmarshal(data, &actionReq); err != nil {
		sendWSError(client, action, "invalid add_kong data")
		return
	}

	gameID := getClientGameID(client)
	playerID := getClientPlayerID(client)

	tileID := int(actionReq.TileId)
	tile := tileFromID(tileID)
	if tile == nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: fmt.Sprintf("無效的牌 ID: %d", tileID),
		})
		return
	}

	state, err := AddKongAction(ctx, gameID, playerID, tile.Type, tile.Value)
	if err != nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: "加槓成功",
	})

	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)
}

func handleDiscardTile(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var actionReq pb.PlayerActionData
	if err := proto.Unmarshal(data, &actionReq); err != nil {
		sendWSError(client, action, "invalid discard_tile data")
		return
	}

	// 從 client metadata 取得 gameID 和 playerID
	gameID := getClientGameID(client)
	playerID := getClientPlayerID(client)

	tile := models.Tile{ID: int(actionReq.TileId)}
	if _, err := DiscardTileAction(ctx, gameID, playerID, tile); err != nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: "出牌成功",
	})

	// 不在此處廣播 WAIT_ACTION 狀態（會導致按鈕閃爍）
	// 由 RunPostDiscard 決定最終狀態後再廣播
	go func() {
		newState, err := RunPostDiscard(context.Background(), gameID)
		if err != nil {
			utils.Error("[WS] RunPostDiscard failed: %v", err)
			return
		}
		if newState != nil {
			syncData := buildSyncStateData(gameID, newState)
			sendProtoBroadcast(client.Hub, "sync_state", syncData)
		}
	}()
}

func handlePlayerAction(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var actionReq pb.PlayerActionData
	if err := proto.Unmarshal(data, &actionReq); err != nil {
		sendWSError(client, action, "invalid PlayerActionData")
		return
	}

	// 從 client metadata 取得 gameID 和 playerID
	gameID := getClientGameID(client)
	playerID := getClientPlayerID(client)

	var state *models.GameState
	var err error

	// action_type: 1=Discard, 2=Chow, 3=Pong, 4=Kong, 5=Hu, 6=Pass
	if actionReq.ActionType == 1 {
		// Discard (出牌) — 建議使用 discard_tile 路由
		tile := models.Tile{ID: int(actionReq.TileId)}
		state, err = DiscardTileAction(ctx, gameID, playerID, tile)
	} else {
		// Declare (Chow/Pong/Kong/Hu/Pass)
		actionStr := "pass"
		switch actionReq.ActionType {
		case 2:
			actionStr = "chow"
		case 3:
			actionStr = "pong"
		case 4:
			actionStr = "kong"
		case 5:
			actionStr = "hu"
		case 6:
			actionStr = "pass"
		}
		state, err = PlayerDeclareAction(ctx, gameID, playerID, actionStr)
	}

	if err != nil {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: "動作成功",
	})

	// 出牌動作：不廣播中間 WAIT_ACTION，由 RunPostDiscard 處理
	if actionReq.ActionType == 1 {
		go func() {
			newState, err := RunPostDiscard(context.Background(), gameID)
			if err != nil {
				utils.Error("[WS] RunPostDiscard failed: %v", err)
				return
			}
			if newState != nil {
				syncData := buildSyncStateData(gameID, newState)
				sendProtoBroadcast(client.Hub, "sync_state", syncData)
			}
		}()
		return
	}

	// 宣告動作 (碰/槓/胡/過)：廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)

	// 如果是宣告動作且結算完畢，觸發後續推進
	if actionReq.ActionType >= 2 && state.Stage != models.StageWaitAction {
		go func() {
			newState, err := RunPostResolve(context.Background(), gameID)
			if err != nil {
				utils.Error("[WS] RunPostResolve failed: %v", err)
				return
			}
			if newState != nil {
				syncData := buildSyncStateData(gameID, newState)
				sendProtoBroadcast(client.Hub, "sync_state", syncData)
			}
		}()
	}
}

func handleNextRound(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid next_round request")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}

	go keepOnline(req.PlayerId)

	state, isComplete, err := NextRound(ctx, gameID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	if isComplete {
		sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
			Success: true,
			Message: "一將結束！遊戲完畢",
		})
		syncData := buildSyncStateData(gameID, state)
		sendProtoBroadcast(client.Hub, "sync_state", syncData)
		return
	}

	// 發牌（NextRound 已設定 stage=DEALING）
	state, err = DealTilesAction(ctx, gameID)
	if err != nil {
		utils.Error("[WS] NextRound DealTilesAction 失敗: %v", err)
		sendWSError(client, action, "發牌失敗: "+err.Error())
		return
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: fmt.Sprintf("進入下一局: %s", state.Round.RoundLabel()),
	})

	// 廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)

	// 如果莊家是 AI，等待前端發牌動畫後再自動出牌
	dealer, ok := state.Players[state.CurrentPlayerID]
	if ok && dealer.IsBot {
		go func() {
			time.Sleep(6 * time.Second)
			if err := ProcessAITurn(context.Background(), gameID, dealer); err != nil {
				utils.Error("[WS] NextRound AI 莊家自動出牌失敗: %v", err)
			}
			if newState, err := LoadGameState(context.Background(), gameID); err == nil {
				syncData := buildSyncStateData(gameID, newState)
				sendProtoBroadcast(client.Hub, "sync_state", syncData)
			}
		}()
	}
}

func handleGetState(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid get_state request")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}

	go keepOnline(req.PlayerId)

	state, err := LoadGameState(ctx, gameID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	syncData := buildSyncStateData(gameID, state)
	sendProtoResponse(client, "sync_state", syncData)
}

func handleGetHands(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid get_hands request")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}

	go keepOnline(req.PlayerId)

	hands, err := GetAllPlayersHands(ctx, gameID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	// 將手牌資訊序列化為 JSON 後透過 PlayerActionRes 回傳
	handsJSON, _ := json.Marshal(hands)
	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: string(handsJSON),
	})
}

func handleGetDeckCount(ctx context.Context, client *websocket.Client, action string, data []byte) {
	var req pb.JoinRoomReq
	if err := proto.Unmarshal(data, &req); err != nil {
		sendWSError(client, action, "invalid get_deck_count request")
		return
	}
	gameID := req.RoomId
	if gameID == "" {
		gameID = "default_room"
	}

	go keepOnline(req.PlayerId)

	count, err := GetDeckCount(ctx, gameID)
	if err != nil {
		sendWSError(client, action, err.Error())
		return
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: fmt.Sprintf("{\"deck_count\": %d}", count),
	})
}

// =====================================
// 幫助函數
// =====================================

// 從 client metadata 取得 gameID
// tileFromID 從 0-based tile ID 取得牌的花色和數值
// ID 0-107: 萬/筒/條 (每 36 張一花色, 每 4 張一數值)
// ID 108-123: 風 (每 4 張一種)
// ID 124-135: 元 (每 4 張一種)
// ID 136-143: 花
func tileFromID(id int) *models.Tile {
	if id < 0 || id > 143 {
		return nil
	}
	group := id / 4
	var tileType models.TileType
	var value int
	switch {
	case group < 9: // 萬 (0-8)
		tileType = models.Wan
		value = group + 1
	case group < 18: // 筒 (9-17)
		tileType = models.Tong
		value = group - 9 + 1
	case group < 27: // 條 (18-26)
		tileType = models.Tiao
		value = group - 18 + 1
	case group < 31: // 風 (27-30)
		tileType = models.Wind
		value = group - 27 + 1
	case group < 34: // 元 (31-33)
		tileType = models.Dragon
		value = group - 31 + 1
	default: // 花 (34-35)
		tileType = models.Flower
		value = id - 136 + 1
	}
	return &models.Tile{ID: id, Type: tileType, Value: value}
}

func getClientGameID(client *websocket.Client) string {
	if v, ok := client.GetMetadata("gameID"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "default_room"
}

// 從 client metadata 取得 playerID
func getClientPlayerID(client *websocket.Client) int {
	if v, ok := client.GetMetadata("playerID"); ok {
		if id, ok := v.(int); ok {
			return id
		}
	}
	return 1
}

// 幫助函數：封裝並發送回應到單一客戶端 (v0.7.1 codec-aware)
// hypgo 的 SendToClient 偵測到 *websocket.Message 時會使用客戶端的 codec 序列化
func sendProtoResponse(client *websocket.Client, action string, data proto.Message) {
	b, err := proto.Marshal(data)
	if err != nil {
		utils.Error("Error marshaling proto data: %v", err)
		return
	}

	msg := &websocket.Message{
		Type: action,
		Data: b,
	}

	client.Hub.SendToClient(client.ID, msg)
}

// 幫助函數：廣播到所有客戶端 (v0.7.1 codec-aware)
// BroadcastMessage 會依每個客戶端的 codec 自動序列化
func sendProtoBroadcast(hub *websocket.Hub, action string, data proto.Message) {
	b, _ := proto.Marshal(data)
	msg := &websocket.Message{
		Type: action,
		Data: b,
	}
	hub.BroadcastMessage(msg)
}

func sendWSError(client *websocket.Client, action string, errorMsg string) {
	utils.Error("[WS Error] Action: %s, Err: %s", action, errorMsg)

	debugMsg := fmt.Sprintf("[DEBUG] Action: %s, Err: %s", action, errorMsg)

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: false,
		Message: debugMsg,
	})
}

// 幫助函數：將 GameState 轉換為 protobuf 定義的 SyncStateData
func buildSyncStateData(gameID string, state *models.GameState) *pb.SyncStateData {
	// 轉換 GameState 狀態名稱
	gameStateStr := string(state.Stage)

	// 將遊戲資訊 (莊家、骰子、結算) 編碼為 JSON
	{
		type GameStatePayload struct {
			Stage           string                     `json:"stage"`
			DealerPlayerID  int                        `json:"dealer_player_id,omitempty"`
			Dice1           int                        `json:"dice1,omitempty"`
			Dice2           int                        `json:"dice2,omitempty"`
			Dice3           int                        `json:"dice3,omitempty"`
			ScoreResults    map[int]models.ScoreResult `json:"score_results,omitempty"`
		}
		payload := GameStatePayload{
			Stage:          gameStateStr,
			DealerPlayerID: state.DealerPlayerID,
			Dice1:          state.Dice.Die1,
			Dice2:          state.Dice.Die2,
		}
		if state.Dice.Die3 > 0 {
			payload.Dice3 = state.Dice.Die3
		}
		if state.Stage == models.StageRoundOver && state.ScoreResults != nil {
			payload.ScoreResults = state.ScoreResults
		}
		if b, err := json.Marshal(payload); err == nil {
			gameStateStr = string(b)
		}
	}

	// 計算剩餘牌數
	deckCount, _ := GetDeckCount(context.Background(), gameID)

	// 計算 lastDiscardedTileId (用 tileID+1 編碼，0 表示無)
	var lastDiscardedTileId int32 = 0
	if state.LastDiscardTile != nil {
		lastDiscardedTileId = int32(state.LastDiscardTile.ID) + 1
	}

	syncData := &pb.SyncStateData{
		RoomId:              gameID,
		CurrentWind:         int32(state.Round.PrevailingWind),
		RemainingTiles:      int32(deckCount),
		CurrentTurnPlayerId: fmt.Sprintf("%d", state.CurrentPlayerID),
		GameState:           gameStateStr,
		LastDiscardedTileId: lastDiscardedTileId,
	}

	// 處理多位贏家的資料傳遞
	if len(state.WinnerIDs) > 0 {
		winnerStrIds := make([]string, len(state.WinnerIDs))
		for i, id := range state.WinnerIDs {
			winnerStrIds[i] = fmt.Sprintf("%d", id)
		}
		syncData.WinnerIds = winnerStrIds
	}

	// 將玩家資料逐一填入
	for p := 1; p <= 4; p++ {
		pInfo := &pb.PlayerInfo{
			Id:   fmt.Sprintf("%d", p),
			Name: fmt.Sprintf("Player %d", p),
			Seat: int32(p),
		}

		if state.Players != nil && state.Players[p].ID != 0 {
			player := state.Players[p]
			// 優先使用暱稱，沒有則用 Name
			if player.Nickname != "" {
				pInfo.Name = player.Nickname
			} else {
				pInfo.Name = player.Name
			}
			pInfo.Score = int32(player.Points)
		}

		// 讀取手牌 (HandTiles)
		hand, err := GetPlayerHand(context.Background(), gameID, p)
		if err == nil {
			for _, t := range hand {
				pInfo.HandTiles = append(pInfo.HandTiles, int32(t.ID))
			}
		}

		// 讀取棄牌 (DiscardedTiles) — 從 Redis 裡讀取
		discards, err := GetPlayerDiscards(context.Background(), gameID, p)
		if err == nil {
			for _, t := range discards {
				pInfo.DiscardedTiles = append(pInfo.DiscardedTiles, int32(t.ID))
			}
		}

		// 讀取副露 (Melds)
		meldsKey := PlayerMeldsKey(gameID, p)
		meldJSONs, _ := service.RedisClient.LRange(context.Background(), meldsKey, 0, -1).Result()
		for _, mj := range meldJSONs {
			var meld models.Meld
			if err := json.Unmarshal([]byte(mj), &meld); err == nil {
				pbMeld := &pb.MeldData{
					Type: int32(meld.Type),
				}
				for _, t := range meld.Tiles {
					pbMeld.Tiles = append(pbMeld.Tiles, int32(t.ID))
				}
				pInfo.Melds = append(pInfo.Melds, pbMeld)
			}
		}

		// 讀取花牌 (Flowers)
		flowersKey := PlayerFlowersKey(gameID, p)
		flowerJSONs, _ := service.RedisClient.LRange(context.Background(), flowersKey, 0, -1).Result()
		for _, fj := range flowerJSONs {
			var tile models.Tile
			if err := json.Unmarshal([]byte(fj), &tile); err == nil {
				pInfo.Flowers = append(pInfo.Flowers, int32(tile.ID))
			}
		}

		syncData.Players = append(syncData.Players, pInfo)
	}

	return syncData
}
