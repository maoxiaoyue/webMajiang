package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisConfig Redis 連線設定
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// RedisClient 全域 Redis 客戶端
var RedisClient *redis.Client

// InitRedis 初始化 Redis 連線
func InitRedis(cfg RedisConfig) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 測試連線
	ctx := context.Background()
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	return nil
}

// CloseRedis 關閉 Redis 連線
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}
