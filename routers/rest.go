package routers

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
	"github.com/maoxiaoyue/hypgo/pkg/router"

	"webmajiang/controllers"
)

// setupRestRoutes 註冊所有 REST API 路由
func setupRestRoutes(r *router.Router) {
	// 基礎路由
	setupBaseRoutes(r)

	// 遊戲路由
	setupGameRoutes(r)

	// 認證路由
	setupAuthRoutes(r)

	// 內部 API (由 gamelobby 呼叫)
	setupInternalRoutes(r)

	// Cocos Creator 遊戲頁面 — 透過 NotFound handler 繞過 hypgo wildcard 路由 bug
	setupGameStaticFallback(r)
}

// setupBaseRoutes 註冊基礎 API 路由
func setupBaseRoutes(r *router.Router) {
	r.GET("/", controllers.GetRoot)
	r.GET("/health", controllers.GetHealth)
	r.POST("/api/client/error", controllers.HandleClientError) // 接收 Cocos 客戶端報錯
}

// serveStaticFile 手動讀取檔案並用 c.Data() 回應，繞過 http.ServeFile 在 HTTP/2 下的空 body 問題
func serveStaticFile(c *hypcontext.Context, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	// 根據副檔名決定 Content-Type
	ext := strings.ToLower(filepath.Ext(path))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Data(http.StatusOK, contentType, data)
}

// setupGameStaticFallback 利用 NotFound handler 處理 /game/* 路徑
// hypgo router 的 catchAll (*filepath) 有 bug，wildChild 沒有正確設定，
// 導致 /game/xxx 路徑永遠不會匹配到 wildcard handler。
// 改用 NotFound handler 攔截 /game/ 開頭的請求。
func setupGameStaticFallback(r *router.Router) {
	gameDir := "build/web-mobile"
	indexPath := filepath.Join(gameDir, "index.html")

	r.NotFound(func(c *hypcontext.Context) {
		urlPath := c.Request.URL.Path

		// 處理 /game 和 /game/ 以及 /game/xxx 的請求
		if urlPath == "/game" || strings.HasPrefix(urlPath, "/game/") {
			// 取得 /game/ 之後的相對路徑
			fp := strings.TrimPrefix(urlPath, "/game")
			fp = strings.TrimPrefix(fp, "/")

			if fp == "" {
				// /game 或 /game/ → serve index.html
				serveStaticFile(c, indexPath)
				return
			}

			// 嘗試讀取對應靜態檔案
			fullPath := filepath.Join(gameDir, fp)
			// 安全檢查：確保路徑不會逃出 gameDir
			absGameDir, _ := filepath.Abs(gameDir)
			absFullPath, _ := filepath.Abs(fullPath)
			if !strings.HasPrefix(absFullPath, absGameDir) {
				c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}

			if _, err := os.Stat(fullPath); err != nil {
				// 檔案不存在 → SPA fallback 到 index.html
				serveStaticFile(c, indexPath)
				return
			}
			serveStaticFile(c, fullPath)
			return
		}

		// 其他路徑 → 真正的 404
		c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	})
}

// setupGameRoutes 註冊遊戲相關路由
func setupGameRoutes(r *router.Router) {
	r.POST("/api/game/start", controllers.StartGameHandler)
}

// setupAuthRoutes 註冊認證相關路由
func setupAuthRoutes(r *router.Router) {
	r.POST("/api/auth/register", controllers.RegisterHandler)
	r.GET("/api/auth/verify", controllers.VerifyEmailHandler)
	r.POST("/api/auth/login", controllers.LoginHandler)
}

// setupInternalRoutes 註冊內部 API 路由 (由 gamelobby 呼叫)
func setupInternalRoutes(r *router.Router) {
	r.POST("/api/internal/create-game", controllers.CreateGameFromLobbyHandler)
}
