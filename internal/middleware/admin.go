package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminRequired 管理员权限中间件
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserIDFromContext(c)
		log.Printf("[AdminRequired] userID=%d, exists=%v", userID, exists)

		if !exists || userID == 0 {
			log.Printf("[AdminRequired] 未认证，拒绝访问")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "未认证",
				"message": "请先登录",
			})
			c.Abort()
			return
		}

		// 检查是否为管理员
		isAdmin, err := IsAdmin(userID)
		log.Printf("[AdminRequired] userID=%d, isAdmin=%v, err=%v", userID, isAdmin, err)

		if err != nil {
			log.Printf("[AdminRequired] 服务器错误: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "服务器错误",
				"message": "无法验证用户权限",
			})
			c.Abort()
			return
		}

		if !isAdmin {
			log.Printf("[AdminRequired] 权限不足，userID=%d", userID)
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "需要管理员权限才能访问此资源",
			})
			c.Abort()
			return
		}

		log.Printf("[AdminRequired] 权限验证通过")
		c.Next()
	}
}

// PermissionRequired 基于权限的中间件
func PermissionRequired(permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserIDFromContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "未认证",
				"message": "请先登录",
			})
			c.Abort()
			return
		}

		// 检查是否有指定权限
		hasPerm, err := HasPermission(userID, permission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "服务器错误",
				"message": "无法验证用户权限",
			})
			c.Abort()
			return
		}

		if !hasPerm {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "权限不足",
				"message": "没有执行此操作的权限",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
