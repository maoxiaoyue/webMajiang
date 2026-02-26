package controllers

import (
	"net/http"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
)

// GetRoot 處理 / 路由（系統狀態檢查）
func GetRoot(c *hypcontext.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"name":    "Web Majiang Game",
		"version": "0.1.0",
		"status":  "running",
	})
}

// GetHealth 處理 /health 路由（健康檢查）
func GetHealth(c *hypcontext.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ok",
	})
}
