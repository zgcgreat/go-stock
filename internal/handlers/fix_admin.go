package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go-stock/backend/db"
	"go-stock/backend/models"
)

// FixAdminRole 修复管理员角色（临时工具接口，生产环境应删除）
func FixAdminRole(c *gin.Context) {
	var adminUser models.User
	
	// 查找 admin 用户
	if err := db.Dao.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "未找到 admin 用户",
			"message": "请先创建 admin 账户",
		})
		return
	}
	
	// 设置 role = admin
	if err := db.Dao.Model(&adminUser).Update("role", "admin").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "更新失败",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "admin 用户已设置为管理员",
		"data": gin.H{
			"id":       adminUser.ID,
			"username": adminUser.Username,
			"role":     adminUser.Role,
		},
	})
}
