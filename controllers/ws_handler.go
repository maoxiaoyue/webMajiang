package controllers

import (
	"context"
	"encoding/json"
	"fmt"
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

		// 如果莊家是 AI，自動觸發莊家出牌
		dealer, ok := state.Players[state.CurrentPlayerID]
		if ok && dealer.IsBot {
			go func() {
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
	state, err := DiscardTileAction(ctx, gameID, playerID, tile)
	if err != nil {
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

	// 廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)

	// 出牌後自動推進遊戲循環 (收集 AI 宣告等)
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

	// 廣播最新狀態
	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)

	// 如果是出牌動作，觸發遊戲循環
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
	}

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

	msg := fmt.Sprintf("進入下一局: %s", state.Round.RoundLabel())
	if isComplete {
		msg = "一將結束！遊戲完畢"
	}

	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: true,
		Message: msg,
	})

	syncData := buildSyncStateData(gameID, state)
	sendProtoBroadcast(client.Hub, "sync_state", syncData)
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

	// 如果處於結算階段且有結果，將台數資訊合併進 GameState string 中 (使用 JSON)
	if state.Stage == models.StageRoundOver && state.ScoreResults != nil {
		type RoundOverPayload struct {
			Stage        string                     `json:"stage"`
			ScoreResults map[int]models.ScoreResult `json:"score_results"`
		}

		payload := RoundOverPayload{
			Stage:        gameStateStr,
			ScoreResults: state.ScoreResults,
		}

		if b, err := json.Marshal(payload); err == nil {
			gameStateStr = string(b)
		}
	}

	// 計算剩餘牌數
	deckCount, _ := GetDeckCount(context.Background(), gameID)

	syncData := &pb.SyncStateData{
		RoomId:              gameID,
		CurrentWind:         int32(state.Round.PrevailingWind),
		RemainingTiles:      int32(deckCount),
		CurrentTurnPlayerId: fmt.Sprintf("%d", state.CurrentPlayerID),
		GameState:           gameStateStr,
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
			pInfo.Name = state.Players[p].Name
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
