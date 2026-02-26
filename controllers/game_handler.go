package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
)

// StartGameHandler 開始新的一將（第一局）
// POST /api/game/start
func StartGameHandler(c *hypcontext.Context) {
	gameID := fmt.Sprintf("majiang_%d", time.Now().UnixNano())
	ctx := context.Background()

	state, err := StartNewGame(ctx, gameID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed to start game",
			"message": err.Error(),
		})
		return
	}

	hands, err := GetAllPlayersHands(ctx, gameID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed to get hands",
			"message": err.Error(),
		})
		return
	}

	remaining, _ := GetDeckCount(ctx, gameID)

	c.JSON(http.StatusOK, map[string]interface{}{
		"message":          "遊戲已開始，發牌完成，已理牌",
		"game_id":          gameID,
		"round":            state.Round.RoundLabel(),
		"round_code":       state.Round.RoundCode(),
		"round_number":     state.Round.RoundNumber(),
		"dice":             state.Dice,
		"dealer_player_id": state.DealerPlayerID,
		"dealer_player":    fmt.Sprintf("player%d", state.DealerPlayerID),
		"deck_remaining":   remaining,
		"players":          hands,
	})
}
