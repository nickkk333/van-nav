package utils

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

// RandomJWTKey 生成一个随机密钥
func RandomJWTKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		logger.LogError("生成随机密钥失败: %v", err)
		return "fallback_secret_key_12345"
	}
	return hex.EncodeToString(bytes)
}

// jwtSecret 签名密钥，可通过环境变量 NAV_JWT_SECRET 固定，否则每次启动随机生成
var jwtSecret []byte

func init() {
	if env := os.Getenv("NAV_JWT_SECRET"); env != "" {
		jwtSecret = []byte(env)
		logger.LogInfo("jwtSecret 来自环境变量 NAV_JWT_SECRET")
		return
	}
	jwtSecret = []byte(RandomJWTKey())
	logger.LogInfo("jwtSecret Setted: %s", jwtSecret)
}

// SignJWT 为用户签名一个 JWT
func SignJWT(user types.User) (string, error) {
	claims := jwt.MapClaims{
		"name": user.Name,
		"id":   user.Id,
		"exp":  time.Now().Add(time.Hour * 24 * 30).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

// SignJWTForAPI 为 API Token 签名一个 JWT
func SignJWTForAPI(tokenName string, tokenId int) (string, error) {
	claims := jwt.MapClaims{
		"name": tokenName,
		"id":   tokenId,
		"exp":  time.Now().Add(time.Hour * 24 * 365 * 100).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

// ParseJWT 解析并校验 JWT
func ParseJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
}

// IsLogin 判断当前请求是否已登录
func IsLogin(c *gin.Context) bool {
	rawToken := c.Request.Header.Get("Authorization")
	if rawToken == "" {
		return false
	}
	token, err := ParseJWT(rawToken)
	return err == nil && token.Valid
}
