package controllers

import (
	"context"
	"fmt"
	"webmajiang/models"
	"webmajiang/models/pb"

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
		var joinReq pb.JoinRoomReq
		if err := proto.Unmarshal(req.Data, &joinReq); err != nil {
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

	// === 玩家動作 (打牌、吃碰槓等) ===
	case "player_action":
		var actionReq pb.PlayerActionData
		if err := proto.Unmarshal(req.Data, &actionReq); err != nil {
			sendWSError(client, action, "invalid PlayerActionData")
			return
		}

		// 註：因為 proto 目前沒有定義額外帶上 room_id，實務上通常會綁定在 client session
		// 這裡做個假設固定 gameID 以符合原有架構
		gameID := "default_room"
		playerID := 1 // 假定對應

		var state *models.GameState
		var err error

		// action_type 1=Discard, 2=Chow, 3=Pong, etc. (參考 proto 定義)
		if actionReq.ActionType == 1 {
			// Discard
			tile := models.Tile{ID: int(actionReq.TileId)}
			state, err = DiscardTileAction(ctx, gameID, playerID, tile)
		} else {
			// Declare (Chow/Pong...)
			// 簡單對應到字串
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

	default:
		fmt.Printf("Unhandled websocket action type: %s\n", action)
	}
}

// 幫助函數：封裝並發送 Protobuf WSMessage 回到單一客戶端
func sendProtoResponse(client *websocket.Client, action string, data proto.Message) {
	b, err := proto.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling proto data:", err)
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

func sendWSError(_ *websocket.Client, action string, errorMsg string) {
	// 簡單封裝錯誤訊息，實務上可自訂 Error Res Proto
	fmt.Printf("[WS Protobuf Error] Action: %s, Err: %s\n", action, errorMsg)
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
