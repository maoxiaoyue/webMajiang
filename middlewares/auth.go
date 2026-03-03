package middlewares

import (
	"context"
	"net/http"
	"strings"

	"webmajiang/models"
	"webmajiang/utils"

	hypcontext "github.com/maoxiaoyue/hypgo/pkg/context"
)

// AuthRequired is a middleware that validates JWT tokens
// and extends the user's logged-in session timeout in KeyDB.
func AuthRequired() hypcontext.HandlerFunc {
	return func(c *hypcontext.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "authorization header required"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Keep user online based on JWT claims
		ctx := context.Background()
		_ = models.KeepUserOnline(ctx, claims.UserID, claims.Username)

		// Set user data in context for subsequent handlers
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
