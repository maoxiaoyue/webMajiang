package routers

import (
	"github.com/maoxiaoyue/hypgo/pkg/router"

	"webmajiang/controllers"
)

// setupRestRoutes 註冊所有 REST API 路由
func setupRestRoutes(r *router.Router) {
	// 基礎路由
	setupBaseRoutes(r)

	// 遊戲路由
	setupGameRoutes(r)
}

// setupBaseRoutes 註冊基礎 API 路由
func setupBaseRoutes(r *router.Router) {
	r.GET("/", controllers.GetRoot)
	r.GET("/health", controllers.GetHealth)
}

// setupGameRoutes 註冊遊戲相關路由
func setupGameRoutes(r *router.Router) {
	r.POST("/api/game/start", controllers.StartGameHandler)
}
