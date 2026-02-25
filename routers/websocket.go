package routers

import (
	"net/http"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
	"github.com/maoxiaoyue/hypgo/pkg/router"
	"github.com/maoxiaoyue/hypgo/pkg/websocket"
)

// setupWebSocketRoutes 註冊 WebSocket 相關路由
func setupWebSocketRoutes(r *router.Router, wsHub *websocket.Hub) {
	r.GET("/ws", wsHub.ServeHTTP)

	r.GET("/ws/stats", func(c *hypcontext.Context) {
		c.JSON(http.StatusOK, wsHub.GetStats())
	})
}
