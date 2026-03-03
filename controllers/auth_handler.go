package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"webmajiang/models"
	"webmajiang/service"
	"webmajiang/utils"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterHandler 處理使用者註冊
func RegisterHandler(c *hypcontext.Context) {
	var req RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid request format"})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "email and password are required"})
		return
	}

	// Hash password
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "internal server error"})
		return
	}

	// Generate random username
	username := fmt.Sprintf("User_%d", time.Now().UnixMilli()%10000)

	ctx := context.Background()

	// Create user
	user, err := models.CreateUser(ctx, username, req.Email, hash)
	if err != nil {
		if err == models.ErrUserExists {
			c.JSON(http.StatusConflict, map[string]interface{}{"error": "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to create user"})
		return
	}

	// Generate verification token
	token, err := utils.GenerateRandomToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to generate token"})
		return
	}

	// Store token in Redis/KeyDB with 1 hour expiration
	redisKey := fmt.Sprintf("verify:%s", token)
	err = service.RedisClient.Set(ctx, redisKey, user.ID, time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to store verification token"})
		return
	}

	// Send verification email
	serverAddr := c.Request.Host
	if serverAddr == "" {
		serverAddr = "localhost:8080"
	}

	// async sending to avoid blocking
	go func() {
		_ = utils.SendVerificationEmail(req.Email, token, serverAddr)
	}()

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Registration successful. Please check your email to verify.",
	})
}

// VerifyEmailHandler 處理信箱驗證
func VerifyEmailHandler(c *hypcontext.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "token is required"})
		return
	}

	ctx := context.Background()
	redisKey := fmt.Sprintf("verify:%s", token)

	// Get user ID from token
	idStr, err := service.RedisClient.Get(ctx, redisKey).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid or expired token"})
		return
	}

	var userID int64
	fmt.Sscanf(idStr, "%d", &userID)

	// Mark user as verified
	if err := models.SetUserVerified(ctx, userID); err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to verify user"})
		return
	}

	// Clean up token
	service.RedisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Email verified successfully. You can now log in.",
	})
}

// LoginHandler 處理使用者登入
func LoginHandler(c *hypcontext.Context) {
	var req LoginRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid request format"})
		return
	}

	ctx := context.Background()

	// Fetch user
	user, err := models.GetUserByEmail(ctx, req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "invalid email or password"})
		return
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "invalid email or password"})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, map[string]interface{}{"error": "please verify your email first"})
		return
	}

	// Add to online users
	err = models.AddUserOnline(ctx, user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to record online status"})
		return
	}

	// Generate JWT string
	token, err := utils.GenerateJWT(user.ID, user.Username, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to generate login token"})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"token":   token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}
