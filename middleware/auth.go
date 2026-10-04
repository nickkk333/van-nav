package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mereith/nav/database"
	"github.com/mereith/nav/utils"
)

// loginRateLimit 登录限流：防暴力破解，单 IP 连续失败 N 次后锁定一段时间
// 计数只统计失败（成功即清零），内存保存（重启清空，单机部署够用）
// 限制：N 次失败后锁定 T 分钟，默认 10 次 / 10 分钟
const (
	loginMaxFails   = 10
	loginLockPeriod = 10 * time.Minute
)

type loginFailRecord struct {
	fails int
	until time.Time
}

var (
	loginFailMu      sync.Mutex
	loginFailRecords = make(map[string]*loginFailRecord)
)

// LoginRateLimit 放在 /api/login 上的限流中间件：失败计数超限时直接 429，不再进 handler 查库
// 计数与清零由 handler 在登录结果确定后调用 LoginFail / LoginSuccess
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		loginFailMu.Lock()
		rec, ok := loginFailRecords[ip]
		if !ok {
			rec = &loginFailRecord{}
			loginFailRecords[ip] = rec
		}
		locked := rec.fails >= loginMaxFails && time.Now().Before(rec.until)
		loginFailMu.Unlock()
		if locked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success":      false,
				"errorMessage": "登录失败次数过多，请稍后再试",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// LoginFail 记录一次登录失败（用户名不存在 / 密码错误都算）：达到上限后锁定一段时间
func LoginFail(c *gin.Context) {
	ip := c.ClientIP()
	loginFailMu.Lock()
	defer loginFailMu.Unlock()
	rec, ok := loginFailRecords[ip]
	if !ok {
		rec = &loginFailRecord{}
		loginFailRecords[ip] = rec
	}
	// 锁定期已过则重新计数，避免一次锁定永久累加
	if !rec.until.IsZero() && time.Now().After(rec.until) {
		rec.fails = 0
		rec.until = time.Time{}
	}
	rec.fails++
	if rec.fails >= loginMaxFails {
		rec.until = time.Now().Add(lockPeriodForFails(rec.fails))
	}
}

// lockPeriodForFails 锁定时间：达到上限后 10 分钟，之后每多失败一次叠加（上限 1 小时），防止持续爆破
func lockPeriodForFails(fails int) time.Duration {
	extra := time.Duration(fails-loginMaxFails) * lockPeriodStep
	if extra > lockPeriodMaxExtra {
		extra = lockPeriodMaxExtra
	}
	return loginLockPeriod + extra
}

const (
	lockPeriodStep     = 10 * time.Minute
	lockPeriodMaxExtra = 50 * time.Minute
)

// LoginSuccess 登录成功后清零该 IP 的失败计数
func LoginSuccess(c *gin.Context) {
	ip := c.ClientIP()
	loginFailMu.Lock()
	defer loginFailMu.Unlock()
	delete(loginFailRecords, ip)
}

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
