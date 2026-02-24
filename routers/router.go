package routers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
	"github.com/maoxiaoyue/hypgo/pkg/router"
	"github.com/maoxiaoyue/hypgo/pkg/websocket"

	"webmajiang/controllers"
)

// Setup 註冊所有路由
func Setup(r *router.Router, wsHub *websocket.Hub) {
	// 基礎路由
	setupBaseRoutes(r)

	// 遊戲路由
	setupGameRoutes(r)

	// WebSocket 路由
	setupWebSocketRoutes(r, wsHub)
}

// setupBaseRoutes 註冊基礎 API 路由
func setupBaseRoutes(r *router.Router) {
	r.GET("/", func(c *hypcontext.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"name":    "Web Majiang Game",
			"version": "0.1.0",
			"status":  "running",
		})
	})

	r.GET("/health", func(c *hypcontext.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
		})
	})
}

// setupGameRoutes 註冊遊戲相關路由
func setupGameRoutes(r *router.Router) {
	// POST /api/game/start - 開始新的一將（第一局）
	// 擲骰子 → 決定莊家 → 建牌堆 → 發牌 → 理牌
	r.POST("/api/game/start", func(c *hypcontext.Context) {
		gameID := fmt.Sprintf("majiang_%d", time.Now().UnixNano())
		ctx := context.Background()

		state, err := controllers.StartNewGame(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to start game",
				"message": err.Error(),
			})
			return
		}

		hands, err := controllers.GetAllPlayersHands(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to get hands",
				"message": err.Error(),
			})
			return
		}

		remaining, _ := controllers.GetDeckCount(ctx, gameID)

		c.JSON(http.StatusOK, map[string]interface{}{
			"message":        "遊戲已開始，發牌完成，已理牌",
			"game_id":        gameID,
			"round":          state.Round.RoundLabel(),
			"round_code":     state.Round.RoundCode(),
			"round_number":   state.Round.RoundNumber(),
			"dice":           state.Dice,
			"dealer_seat":    state.DealerSeat,
			"dealer_player":  fmt.Sprintf("player%d", state.DealerSeat+1),
			"deck_remaining": remaining,
			"players":        hands,
		})
	})

	// POST /api/game/:id/next-round - 進入下一局
	r.POST("/api/game/:id/next-round", func(c *hypcontext.Context) {
		gameID := c.Param("id")
		ctx := context.Background()

		state, isComplete, err := controllers.NextRound(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to advance round",
				"message": err.Error(),
			})
			return
		}

		if isComplete {
			c.JSON(http.StatusOK, map[string]interface{}{
				"message":  "一將結束（北風北已完成）",
				"game_id":  gameID,
				"finished": true,
			})
			return
		}

		hands, err := controllers.GetAllPlayersHands(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to get hands",
				"message": err.Error(),
			})
			return
		}

		remaining, _ := controllers.GetDeckCount(ctx, gameID)

		c.JSON(http.StatusOK, map[string]interface{}{
			"message":        "下一局開始，發牌完成",
			"game_id":        gameID,
			"round":          state.Round.RoundLabel(),
			"round_code":     state.Round.RoundCode(),
			"round_number":   state.Round.RoundNumber(),
			"dealer_seat":    state.DealerSeat,
			"dealer_player":  fmt.Sprintf("player%d", state.DealerSeat+1),
			"deck_remaining": remaining,
			"players":        hands,
		})
	})

	// GET /api/game/:id/state - 查看遊戲狀態（局號、莊家、骰子等）
	r.GET("/api/game/:id/state", func(c *hypcontext.Context) {
		gameID := c.Param("id")
		ctx := context.Background()

		state, err := controllers.LoadGameState(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to load game state",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"game_id":       gameID,
			"round":         state.Round.RoundLabel(),
			"round_code":    state.Round.RoundCode(),
			"round_number":  state.Round.RoundNumber(),
			"dealer_seat":   state.DealerSeat,
			"dealer_player": fmt.Sprintf("player%d", state.DealerSeat+1),
			"dice":          state.Dice,
			"is_started":    state.IsStarted,
			"is_finished":   state.IsFinished,
		})
	})

	// POST /api/game/:id/deal - 對已建立的牌堆進行發牌
	r.POST("/api/game/:id/deal", func(c *hypcontext.Context) {
		gameID := c.Param("id")
		ctx := context.Background()

		// 發牌
		if err := controllers.DealTiles(ctx, gameID); err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to deal tiles",
				"message": err.Error(),
			})
			return
		}

		// 取得發牌結果
		hands, err := controllers.GetAllPlayersHands(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to get hands",
				"message": err.Error(),
			})
			return
		}

		remaining, _ := controllers.GetDeckCount(ctx, gameID)

		c.JSON(http.StatusOK, map[string]interface{}{
			"message":        "發牌完成，已理牌",
			"game_id":        gameID,
			"deck_remaining": remaining,
			"players":        hands,
		})
	})

	// GET /api/game/:id/hands - 查看所有玩家的手牌
	r.GET("/api/game/:id/hands", func(c *hypcontext.Context) {
		gameID := c.Param("id")
		ctx := context.Background()

		hands, err := controllers.GetAllPlayersHands(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to get hands",
				"message": err.Error(),
			})
			return
		}

		remaining, _ := controllers.GetDeckCount(ctx, gameID)

		c.JSON(http.StatusOK, map[string]interface{}{
			"game_id":        gameID,
			"deck_remaining": remaining,
			"players":        hands,
		})
	})

	// GET /api/game/:id/deck/count - 查詢牌堆剩餘張數
	r.GET("/api/game/:id/deck/count", func(c *hypcontext.Context) {
		gameID := c.Param("id")
		ctx := context.Background()

		count, err := controllers.GetDeckCount(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to get deck count",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"game_id":    gameID,
			"deck_count": count,
		})
	})

	// POST /api/game/:id/draw - 摸一張牌
	r.POST("/api/game/:id/draw", func(c *hypcontext.Context) {
		gameID := c.Param("id")
		ctx := context.Background()

		tile, err := controllers.DrawTile(ctx, gameID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error":   "failed to draw tile",
				"message": err.Error(),
			})
			return
		}

		remaining, _ := controllers.GetDeckCount(ctx, gameID)

		c.JSON(http.StatusOK, map[string]interface{}{
			"game_id":   gameID,
			"tile":      tile,
			"tile_name": tile.String(),
			"remaining": remaining,
		})
	})
}

// setupWebSocketRoutes 註冊 WebSocket 相關路由
func setupWebSocketRoutes(r *router.Router, wsHub *websocket.Hub) {
	r.GET("/ws", wsHub.ServeHTTP)

	r.GET("/ws/stats", func(c *hypcontext.Context) {
		c.JSON(http.StatusOK, wsHub.GetStats())
	})
}
