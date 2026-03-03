package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
)

// ClientErrorData 是接收前端錯誤報告的結構體
type ClientErrorData struct {
	Message   string `json:"message"`
	Source    string `json:"source"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Error     string `json:"error"` // Stack Trace 等詳細資訊
	UserAgent string `json:"userAgent"`
	URL       string `json:"url"`
}

// HandleClientError 處理前端回傳的崩潰與例外錯誤
func HandleClientError(c *hypcontext.Context) {
	var req ClientErrorData
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid json payload"})
		return
	}

	// 確保 logs 目錄存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Cannot create log directory: %v\n", err)
	}

	// 準備寫入的格式
	logFilePath := filepath.Join(logDir, "client_error.log")
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Cannot open client error log: %v\n", err)
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to write log"})
		return
	}
	defer f.Close()

	// 格式化紀錄內容
	timestamp := time.Now().Format(time.RFC3339)
	logLine := fmt.Sprintf("[%s] Client Error:\nMsg: %s\nSrc: %s:%d:%d\nStack: %s\nUA: %s\nURL: %s\n================================\n",
		timestamp,
		req.Message,
		req.Source, req.Line, req.Column,
		req.Error,
		req.UserAgent,
		req.URL,
	)

	// 寫入日誌檔案
	if _, err := f.WriteString(logLine); err != nil {
		fmt.Printf("Cannot write to client error log: %v\n", err)
	}

	c.JSON(http.StatusOK, map[string]interface{}{"status": "received"})
}
