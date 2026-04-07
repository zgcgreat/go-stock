package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go-stock/backend/db"
	"go-stock/backend/models"
)

// contextKey 用于在context中存储用户ID
type contextKey string

const userIDKey contextKey = "userID"

// RequireVipRole HTTP中间件：要求用户具有VIP或更高角色
func RequireVipRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := extractUserIDFromRequest(r)
		if userID == 0 {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "需要登录认证")
			return
		}

		// 查询用户信息
		var user models.User
		if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
			writeJSONError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "用户不存在")
			return
		}

		// 检查用户是否激活
		if !user.IsActive {
			writeJSONError(w, http.StatusForbidden, "ACCOUNT_DISABLED", "账户已被禁用")
			return
		}

		// 获取用户角色
		role := GetUserRole(user)

		// 角色是VIP及以上可以直接访问
		if role == RoleVIP || role == RoleAdmin || role == RoleSuperAdmin {
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 检查VIP等级是否有效（在有效期内）
		if user.VipLevel > 0 && user.VipEndAt != nil && !user.VipEndAt.IsZero() {
			if time.Now().Before(*user.VipEndAt) {
				ctx := context.WithValue(r.Context(), userIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		writeJSONError(w, http.StatusForbidden, "VIP_REQUIRED", "此功能需要VIP或更高权限")
		return
	})
}

// RequireAdminRole HTTP中间件：要求用户具有管理员或更高角色
func RequireAdminRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := extractUserIDFromRequest(r)
		if userID == 0 {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "需要登录认证")
			return
		}

		// 查询用户信息
		var user models.User
		if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
			writeJSONError(w, http.StatusUnauthorized, "USER_NOT_FOUND", "用户不存在")
			return
		}

		// 检查用户是否激活
		if !user.IsActive {
			writeJSONError(w, http.StatusForbidden, "ACCOUNT_DISABLED", "账户已被禁用")
			return
		}

		// 获取用户角色
		role := GetUserRole(user)

		// 仅管理员和超级管理员可以访问
		if role != RoleAdmin && role != RoleSuperAdmin {
			writeJSONError(w, http.StatusForbidden, "ADMIN_REQUIRED", "此功能需要管理员权限")
			return
		}

		// 将用户ID存入context
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromHTTPContext 从HTTP请求的context中获取用户ID
func GetUserIDFromHTTPContext(r *http.Request) (uint, bool) {
	userID, ok := r.Context().Value(userIDKey).(uint)
	return userID, ok
}

// extractUserIDFromRequest 从请求头中提取用户ID
func extractUserIDFromRequest(r *http.Request) uint {
	// 从Authorization header中提取token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0
	}

	tokenString := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		tokenString = authHeader
	}

	// 解析token
	claims, err := ParseToken(tokenString)
	if err != nil {
		return 0
	}

	return claims.UserID
}

// writeJSONError 写入JSON错误响应
func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	response := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	_ = json.NewEncoder(w).Encode(response)
}
