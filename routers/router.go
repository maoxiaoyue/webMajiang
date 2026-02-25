package routers

import (
	"github.com/maoxiaoyue/hypgo/pkg/router"
	"github.com/maoxiaoyue/hypgo/pkg/websocket"
)

// Setup 註冊所有路由
func Setup(r *router.Router, wsHub *websocket.Hub) {
	// REST API 路由
	setupRestRoutes(r)

	// WebSocket 路由
	setupWebSocketRoutes(r, wsHub)
}
