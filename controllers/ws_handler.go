package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/maoxiaoyue/hypgo/pkg/websocket"
)

// WSRequest represents the incoming WebSocket message structure payload
type WSRequest struct {
	GameID string `json:"game_id"`
}

// WSResponse represents the outgoing WebSocket response structure
type WSResponse struct {
	Type    string      `json:"type"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HandleWebSocketMessage acts as the main router for incoming WebSocket messages
func HandleWebSocketMessage(client *websocket.Client, msg *websocket.Message) {
	ctx := context.Background()

	var req WSRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil && msg.Type != "ping" {
		sendWSError(client, msg.Type+"_response", "invalid payload format")
		return
	}

	gameID := req.GameID

	switch msg.Type {
	case "next_round":
		state, isComplete, err := NextRound(ctx, gameID)
		if err != nil {
			sendWSError(client, "next_round_response", err.Error())
			return
		}

		if isComplete {
			client.Hub.SendToClient(client.ID, WSResponse{
				Type:    "next_round_response",
				Message: "一將結束（北風北已完成）",
				Data: map[string]interface{}{
					"game_id":  gameID,
					"finished": true,
				},
			})
			return
		}

		hands, err := GetAllPlayersHands(ctx, gameID)
		if err != nil {
			sendWSError(client, "next_round_response", "failed to get hands: "+err.Error())
			return
		}

		remaining, _ := GetDeckCount(ctx, gameID)

		client.Hub.SendToClient(client.ID, WSResponse{
			Type:    "next_round_response",
			Message: "下一局開始，發牌完成",
			Data: map[string]interface{}{
				"game_id":          gameID,
				"round":            state.Round.RoundLabel(),
				"round_code":       state.Round.RoundCode(),
				"round_number":     state.Round.RoundNumber(),
				"dealer_player_id": state.DealerPlayerID,
				"dealer_player":    fmt.Sprintf("player%d", state.DealerPlayerID),
				"deck_remaining":   remaining,
				"players":          hands,
			},
		})

	case "get_state":
		state, err := LoadGameState(ctx, gameID)
		if err != nil {
			sendWSError(client, "get_state_response", err.Error())
			return
		}

		client.Hub.SendToClient(client.ID, WSResponse{
			Type: "get_state_response",
			Data: map[string]interface{}{
				"game_id":          gameID,
				"round":            state.Round.RoundLabel(),
				"round_code":       state.Round.RoundCode(),
				"round_number":     state.Round.RoundNumber(),
				"dealer_player_id": state.DealerPlayerID,
				"dealer_player":    fmt.Sprintf("player%d", state.DealerPlayerID),
				"dice":             state.Dice,
				"is_started":       state.IsStarted,
				"is_finished":      state.IsFinished,
			},
		})

	case "roll_positions":
		state, err := RollPositions(ctx, gameID)
		if err != nil {
			sendWSError(client, "roll_positions_response", err.Error())
			return
		}
		client.Hub.SendToClient(client.ID, WSResponse{
			Type:    "roll_positions_response",
			Message: "座位決定完成",
			Data: map[string]interface{}{
				"game_id": gameID,
				"dice":    state.Dice,
				"stage":   state.Stage,
			},
		})

	case "roll_dealer":
		state, err := RollDealer(ctx, gameID)
		if err != nil {
			sendWSError(client, "roll_dealer_response", err.Error())
			return
		}
		client.Hub.SendToClient(client.ID, WSResponse{
			Type:    "roll_dealer_response",
			Message: "莊家決定完成",
			Data: map[string]interface{}{
				"game_id":          gameID,
				"dice":             state.Dice,
				"dealer_player_id": state.DealerPlayerID,
				"stage":            state.Stage,
			},
		})

	case "deal_tiles":
		state, err := DealTilesAction(ctx, gameID)
		if err != nil {
			sendWSError(client, "deal_tiles_response", "failed to deal tiles: "+err.Error())
			return
		}

		hands, err := GetAllPlayersHands(ctx, gameID)
		if err != nil {
			sendWSError(client, "deal_tiles_response", "failed to get hands: "+err.Error())
			return
		}

		remaining, _ := GetDeckCount(ctx, gameID)

		client.Hub.SendToClient(client.ID, WSResponse{
			Type:    "deal_tiles_response",
			Message: "發牌完成，已理牌",
			Data: map[string]interface{}{
				"game_id":           gameID,
				"deck_remaining":    remaining,
				"players":           hands,
				"stage":             state.Stage,
				"current_player_id": state.CurrentPlayerID,
			},
		})

	case "discard_tile":
		sendWSError(client, "discard_tile_response", "discard tile not fully implemented yet")
		// To be fully implemented in game.go DiscardTile action

	case "player_action":
		sendWSError(client, "player_action_response", "player action not fully implemented yet")
		// To be fully implemented in game.go PlayerAction action

	case "get_hands":
		hands, err := GetAllPlayersHands(ctx, gameID)
		if err != nil {
			sendWSError(client, "get_hands_response", err.Error())
			return
		}

		remaining, _ := GetDeckCount(ctx, gameID)

		client.Hub.SendToClient(client.ID, WSResponse{
			Type: "get_hands_response",
			Data: map[string]interface{}{
				"game_id":        gameID,
				"deck_remaining": remaining,
				"players":        hands,
			},
		})

	case "get_deck_count":
		count, err := GetDeckCount(ctx, gameID)
		if err != nil {
			sendWSError(client, "get_deck_count_response", err.Error())
			return
		}

		client.Hub.SendToClient(client.ID, WSResponse{
			Type: "get_deck_count_response",
			Data: map[string]interface{}{
				"game_id":    gameID,
				"deck_count": count,
			},
		})

	case "draw_tile":
		tile, err := DrawTile(ctx, gameID)
		if err != nil {
			sendWSError(client, "draw_tile_response", err.Error())
			return
		}

		remaining, _ := GetDeckCount(ctx, gameID)

		client.Hub.SendToClient(client.ID, WSResponse{
			Type: "draw_tile_response",
			Data: map[string]interface{}{
				"game_id":   gameID,
				"tile":      tile,
				"tile_name": tile.String(),
				"remaining": remaining,
			},
		})

	default:
		// Optional: handle unknown message types
		fmt.Printf("Unhandled websocket message type: %s\n", msg.Type)
	}
}

func sendWSError(client *websocket.Client, responseType string, errorMsg string) {
	client.Hub.SendToClient(client.ID, WSResponse{
		Type:  responseType,
		Error: errorMsg,
	})
}
