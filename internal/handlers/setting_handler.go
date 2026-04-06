package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/internal/middleware"
)

// UserSetting 用户设置模型
type UserSetting struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UserID      uint      `gorm:"index" json:"userId"`
	Type        string    `json:"type"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
}

func (UserSetting) TableName() string {
	return "user_settings"
}

// GetUserSettings 获取用户设置列表
func GetUserSettings(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	var total int64
	var settings []UserSetting

	query := db.Dao.Where("user_id = ?", userID)

	if key := c.Query("key"); key != "" {
		query = query.Where("key LIKE ?", "%"+key+"%")
	}

	if typeVal := c.Query("type"); typeVal != "" {
		query = query.Where("type LIKE ?", "%"+typeVal+"%")
	}

	if description := c.Query("description"); description != "" {
		query = query.Where("description LIKE ?", "%"+description+"%")
	}

	query.Model(&UserSetting{}).Count(&total)

	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&settings).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch user settings",
			"message": "获取用户设置失败",
		})
		return
	}

	totalPages := total / int64(pageSize)
	if total%int64(pageSize) > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       settings,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// SetUserSetting 设置用户设置
func SetUserSetting(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var settingReq struct {
		Type        string `json:"type" binding:"required"`
		Key         string `json:"key" binding:"required"`
		Value       string `json:"value" binding:"required"`
		Description string `json:"description,omitempty"`
	}

	if err := c.ShouldBindJSON(&settingReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingSetting UserSetting
	result := db.Dao.Where("key = ? AND user_id = ?", settingReq.Key, userID).First(&existingSetting)
	if result.Error == nil {
		err := db.Dao.Model(&existingSetting).Where("id = ?", existingSetting.ID).Updates(UserSetting{
			Type:        settingReq.Type,
			Value:       settingReq.Value,
			Description: settingReq.Description,
		}).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update user setting",
				"message": "更新用户设置失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "User setting updated successfully",
			"data": gin.H{
				"type":        settingReq.Type,
				"key":         settingReq.Key,
				"value":       settingReq.Value,
				"description": settingReq.Description,
			},
		})
		return
	}

	newSetting := &UserSetting{
		UserID:      userID,
		Type:        settingReq.Type,
		Key:         settingReq.Key,
		Value:       settingReq.Value,
		Description: settingReq.Description,
	}

	if err := db.Dao.Create(newSetting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user setting",
			"message": "创建用户设置失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "User setting created successfully",
		"data":    newSetting,
	})
}

// UpdateUserSetting 更新用户设置
func UpdateUserSetting(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Setting key is required",
			"message": "设置键不能为空",
		})
		return
	}

	var updateReq struct {
		Type        *string `json:"type"`
		Value       *string `json:"value"`
		Description *string `json:"description"`
	}

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var setting UserSetting
	result := db.Dao.Where("key = ? AND user_id = ?", key, userID).First(&setting)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User setting not found",
			"message": "未找到对应的用户设置",
		})
		return
	}

	updates := make(map[string]interface{})

	if updateReq.Type != nil {
		updates["type"] = *updateReq.Type
	}

	if updateReq.Value != nil {
		updates["value"] = *updateReq.Value
	}

	if updateReq.Description != nil {
		updates["description"] = *updateReq.Description
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No fields to update",
			"message": "没有可更新的字段",
		})
		return
	}

	err := db.Dao.Model(&setting).Where("id = ?", setting.ID).Updates(updates).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update user setting",
			"message": "更新用户设置失败",
		})
		return
	}

	db.Dao.Where("id = ?", setting.ID).First(&setting)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "User setting updated successfully",
		"data":    setting,
	})
}

// DeleteUserSetting 删除用户设置
func DeleteUserSetting(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Setting key is required",
			"message": "设置键不能为空",
		})
		return
	}

	var setting UserSetting
	result := db.Dao.Where("key = ? AND user_id = ?", key, userID).First(&setting)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User setting not found",
			"message": "未找到对应的用户设置",
		})
		return
	}

	if err := db.Dao.Delete(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete user setting",
			"message": "删除用户设置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "User setting deleted successfully",
	})
}

// GetAppConfig 获取应用配置（对应 Wails 的 GetConfig）
func GetAppConfig(c *gin.Context) {
	config := data.GetSettingConfig()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    config,
	})
}

// UpdateAppConfig 更新应用配置（对应 Wails 的 UpdateConfig）
func UpdateAppConfig(c *gin.Context) {
	var config data.SettingConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	result := data.UpdateConfig(&config)
	_ = result // result 为 "保存成功！" 等提示文字，非错误信息

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "配置更新成功",
		"data":    config,
	})
}

// HealthCheck 健康检查
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "go-stock-web-api",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
