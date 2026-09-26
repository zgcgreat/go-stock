package middleware

import (
	"crypto/rand"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"go-stock/backend/models"
)

// jwtSecret 存储 JWT 签名密钥。
// 生产环境必须通过环境变量 GO_STOCK_JWT_SECRET 配置固定密钥；
// 未配置时回退为每次启动随机生成（重启后已签发 token 失效，仅用于本地开发）。
var jwtSecret []byte

func init() {
	if secret := os.Getenv("GO_STOCK_JWT_SECRET"); secret != "" {
		jwtSecret = []byte(secret)
		return
	}
	log.Println("[WARN] GO_STOCK_JWT_SECRET 未设置，使用随机临时密钥（重启后已签发 token 将失效）。生产部署请配置固定密钥。")
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// 极端兜底（不安全，仅防止 panic）
		b = []byte(time.Now().Format("20060102150405.000000000"))
	}
	jwtSecret = b
}

// ---------------------------------------------------------------------------
// 令牌吊销（登出/改密后使旧 token 失效）
// 轻量级内存黑名单：存储 jti -> 过期时间；过期后自动清理。
// 多实例部署建议替换为共享存储（Redis），当前单实例场景足够。
// ---------------------------------------------------------------------------
var revokedTokens sync.Map // map[jti]string(过期时间RFC3339)

// RevokeToken 将指定 jti 的 token 加入黑名单，直到其原始过期时间。
func RevokeToken(jti string, expiresAt time.Time) {
	if jti == "" {
		return
	}
	revokedTokens.Store(jti, expiresAt)
}

// IsTokenRevoked 判断 token 是否已吊销（已过期且仍缓存的也会返回 false 以避免无限增长）。
func IsTokenRevoked(jti string) bool {
	if jti == "" {
		return false
	}
	v, ok := revokedTokens.Load(jti)
	if !ok {
		return false
	}
	exp, ok := v.(time.Time)
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		revokedTokens.Delete(jti)
		return false
	}
	return true
}

// generateJTI 生成随机 jti，避免 token 重放与吊销混淆。
func generateJTI() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	const hexChars = "0123456789abcdef"
	out := make([]byte, 32)
	for i := 0; i < 32; i++ {
		out[i] = hexChars[b[i%16]&0x0f]
	}
	return string(out)
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "go-stock-api",
			ID:        generateJTI(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// 校验吊销黑名单
	if IsTokenRevoked(claims.ID) {
		return nil, errors.New("token has been revoked")
	}

	return claims, nil
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authorization header required",
				"message": "请先登录",
			})
			c.Abort()
			return
		}

		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			tokenString = authHeader
		}

		claims, err := ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid token",
				"message": "无效的认证令牌",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// AuthOptional 是可选认证中间件，用于 Web 模式。
// 有 token 时解析用户信息；无 token 时使用匿名用户（ID=0），不拦截请求。
func AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Set("user_id", uint(0))
			c.Set("username", "anonymous")
			c.Next()
			return
		}

		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			tokenString = authHeader
		}

		claims, err := ParseToken(tokenString)
		if err != nil {
			// token 无效时降级为匿名用户，不拦截
			c.Set("user_id", uint(0))
			c.Set("username", "anonymous")
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

func GetUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	if id, ok := userID.(float64); ok {
		return uint(id), true
	}

	if id, ok := userID.(uint); ok {
		return id, true
	}

	if id, ok := userID.(int); ok {
		return uint(id), true
	}

	return 0, false
}

func GetUsernameFromContext(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}

	if name, ok := username.(string); ok {
		return name, true
	}

	return "", false
}

// GetUserIDFromHTTPContext 从标准库 *http.Request 的 Authorization 头解析用户ID。
// 供非 gin 的 net/http 处理器（如桌面端 Wails 桥接的 Handle* 系列）使用。
func GetUserIDFromHTTPContext(r *http.Request) (uint, bool) {
	auth := r.Header.Get("Authorization")
	tokenStr := ""
	if strings.HasPrefix(auth, "Bearer ") {
		tokenStr = strings.TrimPrefix(auth, "Bearer ")
	} else if auth != "" {
		tokenStr = auth
	}
	if tokenStr == "" {
		return 0, false
	}
	claims, err := ParseToken(tokenStr)
	if err != nil {
		return 0, false
	}
	return claims.UserID, true
}
