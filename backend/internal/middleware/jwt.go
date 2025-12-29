package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"p2p-chat-app/backend/pkg/utils"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization"})
			return
		}
		claims, err := utils.ParseToken(parts[1], secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userId", claims.UserID)
		c.Next()
	}
}
