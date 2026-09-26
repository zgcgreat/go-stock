package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LoginRateLimit 针对认证接口的简易内存限流/锁容，防御暴力破解与用户枚举。
// 策略：同一客户端 IP 在窗口内失败次数超过上限即拒绝，直到窗口结束。
// 多实例部署建议替换为共享计数器（如 Redis），单实例场景足够。
const (
	rateLimitMaxAttempts = 8
	rateLimitWindow      = 15 * time.Minute
)

type loginAttempt struct {
	count     int
	windowEnd time.Time
}

var (
	loginAttemptMu sync.Mutex
	loginAttempts  = map[string]*loginAttempt{}
)

// LoginRateLimit 限流中间件（按客户端 IP）。
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		loginAttemptMu.Lock()
		now := time.Now()
		a, ok := loginAttempts[ip]
		if !ok || now.After(a.windowEnd) {
			loginAttempts[ip] = &loginAttempt{count: 0, windowEnd: now.Add(rateLimitWindow)}
			a = loginAttempts[ip]
		}
		if a.count >= rateLimitMaxAttempts {
			loginAttemptMu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "RATE_LIMITED",
				"message": "尝试次数过多，请稍后再试",
			})
			return
		}
		loginAttemptMu.Unlock()
		c.Next()
	}
}

// RecordLoginFailure 记录一次失败尝试（登录/注册校验失败时调用）。
func RecordLoginFailure(ip string) {
	loginAttemptMu.Lock()
	defer loginAttemptMu.Unlock()
	now := time.Now()
	a, ok := loginAttempts[ip]
	if !ok || now.After(a.windowEnd) {
		loginAttempts[ip] = &loginAttempt{count: 1, windowEnd: now.Add(rateLimitWindow)}
		return
	}
	a.count++
}

// ResetLoginAttempts 成功后清零（防止正常用户被误伤）。
func ResetLoginAttempts(ip string) {
	loginAttemptMu.Lock()
	defer loginAttemptMu.Unlock()
	delete(loginAttempts, ip)
}
