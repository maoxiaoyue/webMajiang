package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"webmajiang/models"
	"webmajiang/models/pb"
	"webmajiang/utils"

	"github.com/maoxiaoyue/hypgo/pkg/websocket"
	"google.golang.org/protobuf/proto"
)

// HandleWebSocketMessage acts as the main router for incoming WebSocket messages (Now using Protobuf)
func HandleWebSocketMessage(client *websocket.Client, msg *websocket.Message) {
	ctx := context.Background()

	// 1. 先解析最外層的 WSMessage (Protobuf)
	var req pb.WSMessage
	if err := proto.Unmarshal(msg.Data, &req); err != nil {
		sendWSError(client, "ParseError", "invalid protobuf payload")
		return
	}

	action := req.Action

	switch action {
	// === 玩家加入房間 ===
	case "join_room":
		handleJoinRoom(ctx, client, action, req.Data)

	// === 決定座位 (擲骰子) ===
	case "roll_positions":
		handleRollPositions(ctx, client, action, req.Data)

	// === 決定莊家 (擲骰子) ===
	case "roll_dealer":
		handleRollDealer(ctx, client, action, req.Data)

	// === 觸發發牌 ===
	case "deal_tiles":
		handleDealTiles(ctx, client, action, req.Data)

	// === 手牌排序 ===
	case "sort_hand":
		handleSortHand(ctx, client, action, req.Data)

	// === 玩家摸牌 ===
	case "draw_tile":
		handleDrawTile(ctx, client, action, req.Data)

	// === 玩家出牌 ===
	case "discard_tile":
		handleDiscardTile(ctx, client, action, req.Data)

	// === 玩家宣告 (吃/碰/槓/胡/放棄) ===
	case "player_action":
		handlePlayerAction(ctx, client, action, req.Data)

	// === 進入下一局 ===
	case "next_round":
		handleNextRound(ctx, client, action, req.Data)

	// === 查詢目前遊戲狀態 ===
	case "get_state":
		handleGetState(ctx, client, action, req.Data)

	// === 查詢各家手牌 ===
	case "get_hands":
		handleGetHands(ctx, client, action, req.Data)

	// === 查詢牌堆剩餘數量 ===
	case "get_deck_count":
		handleGetDeckCount(ctx, client, action, req.Data)

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

	// 假設加入成功，回覆
	res := &pb.JoinRoomRes{
		Success: true,
		Message: "加入成功",
	}
	sendProtoResponse(client, action+"_res", res)

	// 順便發送一次全房狀態同步
	state, err := LoadGameState(ctx, gameID)
	if err == nil {
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

	// 從 client session / request 取得 gameID 和 playerID
	// TODO: 實務上應從 client session 中取得，目前仍需前端帶上
	gameID := "default_room"
	playerID := 1

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

	// TODO: 從 client session 中取得 gameID 和 playerID
	gameID := "default_room"
	playerID := 1

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

// 幫助函數：封裝並發送 Protobuf WSMessage 回到單一客戶端
func sendProtoResponse(client *websocket.Client, action string, data proto.Message) {
	b, err := proto.Marshal(data)
	if err != nil {
		utils.Error("Error marshaling proto data: %v", err)
		return
	}

	msg := &pb.WSMessage{
		Action: action,
		Data:   b,
	}

	outBytes, _ := proto.Marshal(msg)

	// 發送二進位資料
	client.Hub.SendToClient(client.ID, outBytes)
}

// 幫助函數：封裝並廣播 Protobuf WSMessage
func sendProtoBroadcast(hub *websocket.Hub, action string, data proto.Message) {
	b, _ := proto.Marshal(data)
	msg := &pb.WSMessage{
		Action: action,
		Data:   b,
	}
	outBytes, _ := proto.Marshal(msg)
	hub.Broadcast(outBytes)
}

func sendWSError(client *websocket.Client, action string, errorMsg string) {
	utils.Error("[WS Protobuf Error] Action: %s, Err: %s", action, errorMsg)

	// Send to frontend as [DEBUG]
	debugMsg := fmt.Sprintf("[DEBUG] Action: %s, Err: %s", action, errorMsg)

	// 簡單封裝錯誤訊息回傳給前端
	sendProtoResponse(client, action+"_res", &pb.PlayerActionRes{
		Success: false,
		Message: debugMsg,
	})
}

// 幫助函數：將 GameState 轉換為 protobuf 定義的 SyncStateData
func buildSyncStateData(gameID string, state *models.GameState) *pb.SyncStateData {
	syncData := &pb.SyncStateData{
		RoomId:              gameID,
		CurrentWind:         int32(state.Round.PrevailingWind),
		CurrentTurnPlayerId: fmt.Sprintf("%d", state.CurrentPlayerID),
		GameState:           string(state.Stage),
	}

	// 將玩家資料逐一填入
	// for _, p := range state.Players ...

	return syncData
}
