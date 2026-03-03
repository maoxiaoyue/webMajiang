package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"webmajiang/service"

	"github.com/redis/go-redis/v9"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// User 代表系統中的使用者
type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	IsVerified   bool   `json:"is_verified"`
}

// CreateUser 建立新使用者，回傳新建的使用者
func CreateUser(ctx context.Context, username, email, passwordHash string) (*User, error) {
	// 檢查信箱是否已存在
	exists, err := service.RedisClient.HExists(ctx, "user:emails", email).Result()
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}

	// 產生新的 ID
	id, err := service.RedisClient.Incr(ctx, "user:id_counter").Result()
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		IsVerified:   false,
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	// 使用 Transaction 來確保原子性
	pipe := service.RedisClient.TxPipeline()
	pipe.HSet(ctx, "user:emails", email, id)
	pipe.Set(ctx, fmt.Sprintf("user:info:%d", id), userJSON, 0)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail 根據 email 尋找使用者
func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	// 取得使用者 ID
	idStr, err := service.RedisClient.HGet(ctx, "user:emails", email).Result()
	if err == redis.Nil {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, err
	}

	return GetUserByID(ctx, id)
}

// GetUserByID 根據 ID 尋找使用者
func GetUserByID(ctx context.Context, id int64) (*User, error) {
	userJSON, err := service.RedisClient.Get(ctx, fmt.Sprintf("user:info:%d", id)).Result()
	if err == redis.Nil {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	var user User
	err = json.Unmarshal([]byte(userJSON), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// SetUserVerified 標示使用者 email 已驗證
func SetUserVerified(ctx context.Context, id int64) error {
	user, err := GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	user.IsVerified = true
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return service.RedisClient.Set(ctx, fmt.Sprintf("user:info:%d", id), userJSON, 0).Err()
}

// AddUserOnline 將使用者加入 user:online 並設定 10 分鐘過期
func AddUserOnline(ctx context.Context, id int64, username string) error {
	member := fmt.Sprintf("%d:%s", id, username)
	key := "user:online"

	// 將使用者加入 Set 中
	err := service.RedisClient.SAdd(ctx, key, member).Err()
	if err != nil {
		return err
	}

	// 呼叫 KeyDB 專屬指令 EXPIREMEMBER
	// 語法: EXPIREMEMBER key seconds member
	err = service.RedisClient.Do(ctx, "EXPIREMEMBER", key, 600, member).Err()
	return err
}

// KeepUserOnline 延長使用者在 user:online 的有效時間
func KeepUserOnline(ctx context.Context, id int64, username string) error {
	// 如果已經存在於 Set，重新設定超時即可，SADD 沒害處但也可以直接調用 EXPIREMEMBER
	member := fmt.Sprintf("%d:%s", id, username)
	key := "user:online"

	// 先嘗試延遲過期
	err := service.RedisClient.Do(ctx, "EXPIREMEMBER", key, 600, member).Err()
	if err != nil {
		// 如果出現 error 可能是還沒加入 Set，順便 SADD 一次
		service.RedisClient.SAdd(ctx, key, member)
		err = service.RedisClient.Do(ctx, "EXPIREMEMBER", key, 600, member).Err()
	}
	return err
}
