package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"webmajiang/models"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
)

// startGameRequest 開始遊戲請求
type startGameRequest struct {
	GameType int `json:"game_type"` // 13 或 16，預設 16
}

// StartGameHandler 開始新的一將（第一局）
// POST /api/game/start
// Body: {"game_type": 13} 或 {"game_type": 16}
func StartGameHandler(c *hypcontext.Context) {
	gameID := fmt.Sprintf("majiang_%d", time.Now().UnixNano())
	ctx := context.Background()

	// 解析請求參數
	var req startGameRequest
	if err := c.BindJSON(&req); err != nil {
		// 解析失敗時使用預設值
		req.GameType = 16
	}

	// 轉換遊戲類型
	gameType := models.GameType(req.GameType)
	if gameType != models.GameType13 && gameType != models.GameType16 {
		gameType = models.GameType16
	}

	state, err := StartNewGame(ctx, gameID, gameType)
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
		"game_type":    int(gameType),
		"stage":        state.Stage,
		"round":        state.Round.RoundLabel(),
		"round_code":   state.Round.RoundCode(),
		"round_number": state.Round.RoundNumber(),
	})
}
