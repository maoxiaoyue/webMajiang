package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maoxiaoyue/hypgo/pkg/config"
	"github.com/maoxiaoyue/hypgo/pkg/logger"
	"github.com/maoxiaoyue/hypgo/pkg/server"
	"github.com/maoxiaoyue/hypgo/pkg/websocket"

	"webmajiang/controllers"
	"webmajiang/routers"
	"webmajiang/service"
)

// AppConfig 應用程式配置（包含 Redis）
type AppConfig struct {
	Redis service.RedisConfig `yaml:"redis"`
}

func main() {
	// 載入配置
	cfg := &config.Config{}
	loader := config.NewConfigLoader("config")
	if err := loader.Load("config/config.yaml", cfg); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 載入自訂配置（Redis）
	appCfg := &AppConfig{}
	if err := loader.Load("config/config.yaml", appCfg); err != nil {
		fmt.Printf("Failed to load app config: %v\n", err)
		os.Exit(1)
	}

	// 初始化 Logger
	log, err := logger.New(
		cfg.Logger.Level,
		cfg.Logger.Output,
		nil,
		cfg.Logger.ColorEnabled,
	)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// 初始化 Redis
	if err := service.InitRedis(appCfg.Redis); err != nil {
		log.Fatal("Failed to connect to Redis: %v", err)
	}
	defer service.CloseRedis()
	log.Info("Redis connected at %s", appCfg.Redis.Addr)

	// 建立伺服器
	srv := server.New(cfg, log)
	r := srv.Router()

	// WebSocket Hub
	wsHub := websocket.NewHub(log, websocket.DefaultConfig)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go wsHub.Run(ctx)

	// 設置 WebSocket 回調
	wsHub.SetCallbacks(
		func(client *websocket.Client) {
			log.Info("Player connected: %s", client.ID)
		},
		func(client *websocket.Client) {
			log.Info("Player disconnected: %s", client.ID)
		},
		func(client *websocket.Client, msg *websocket.Message) {
			log.Debug("Message from %s: type=%s", client.ID, msg.Type)
			controllers.HandleWebSocketMessage(client, msg)
		},
	)

	// 註冊路由
	routers.Setup(r, wsHub)

	// 靜態檔案
	srv.Static("/static", "static")

	// 啟動伺服器
	log.Info("Starting Web Majiang Game server on %s", cfg.Server.Addr)
	if err := srv.Start(); err != nil {
		log.Fatal("Server failed to start: %v", err)
	}
}
