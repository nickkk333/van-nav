package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mereith/nav/database"
	"github.com/mereith/nav/utils"
)

// JWTMiddleware 校验 JWT，同时兼容之前签发的 API Token
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken := c.Request.Header.Get("Authorization")
		if rawToken == "" {
			unauthorized(c)
			return
		}

		// API Token 直接放行
		if database.HasApiToken(rawToken) {
			c.Set("username", "apiToken")
			c.Set("uid", 1)
			c.Next()
			return
		}

		// 解析并校验 JWT
		token, err := utils.ParseJWT(rawToken)
		if err != nil {
			unauthorized(c)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			unauthorized(c)
			return
		}
		c.Set("username", claims["name"])
		c.Set("uid", claims["id"])
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"success":      false,
		"errorMessage": "未登录",
	})
	c.Abort()
}
