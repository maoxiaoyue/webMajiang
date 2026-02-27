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

	c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "遊戲初始化完成，等待進入 WebSocket 進行後續階段",
		"game_id":      gameID,
		"stage":        state.Stage,
		"round":        state.Round.RoundLabel(),
		"round_code":   state.Round.RoundCode(),
		"round_number": state.Round.RoundNumber(),
	})
}
