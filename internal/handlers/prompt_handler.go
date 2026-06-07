package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go-stock/backend/db"
	"go-stock/backend/models"
	"go-stock/internal/middleware"
)

// GetPromptTemplates 获取提示模板列表
func GetPromptTemplates(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

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
	var templates []models.PromptTemplate

	query := db.Dao

	query = query.Where("user_id = ? OR is_public = ?", userID, true)

	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if templateType := c.Query("type"); templateType != "" {
		query = query.Where("type = ?", templateType)
	}

	if content := c.Query("content"); content != "" {
		query = query.Where("content LIKE ?", "%"+content+"%")
	}

	isPublicStr := c.Query("isPublic")
	if isPublicStr != "" {
		isPublic, err := strconv.ParseBool(isPublicStr)
		if err == nil {
			query = query.Where("is_public = ?", isPublic)
		}
	}

	query.Model(&models.PromptTemplate{}).Count(&total)

	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&templates).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch prompt templates",
			"message": "获取提示模板失败",
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
			"list":       templates,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// CreatePromptTemplate 创建提示模板
func CreatePromptTemplate(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var templateReq struct {
		Name     string `json:"name" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Type     string `json:"type" binding:"required"`
		IsPublic bool   `json:"isPublic"`
	}

	if err := c.ShouldBindJSON(&templateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingTemplate models.PromptTemplate
	result := db.Dao.Where("name = ? AND user_id = ?", templateReq.Name, userID).First(&existingTemplate)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    1,
			"message": "该模板名称已存在，请更换名称后重试",
		})
		return
	}

	newTemplate := &models.PromptTemplate{
		UserID:   userID,
		Name:     templateReq.Name,
		Content:  templateReq.Content,
		Type:     templateReq.Type,
		IsPublic: templateReq.IsPublic,
	}

	if err := db.Dao.Create(newTemplate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create prompt template",
			"message": "创建提示模板失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "Prompt template created successfully",
		"data":    newTemplate,
	})
}

// UpdatePromptTemplate 更新提示模板
func UpdatePromptTemplate(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID format",
			"message": "无效的ID格式",
		})
		return
	}

	var templateReq struct {
		Name     *string `json:"name"`
		Content  *string `json:"content"`
		Type     *string `json:"type"`
		IsPublic *bool   `json:"isPublic"`
	}

	if err := c.ShouldBindJSON(&templateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var template models.PromptTemplate
	result := db.Dao.Where("id = ? AND user_id = ?", uint(id), userID).First(&template)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Prompt template not found",
			"message": "未找到对应的提示模板",
		})
		return
	}

	updateFields := make(map[string]interface{})

	if templateReq.Name != nil {
		updateFields["name"] = *templateReq.Name
	}

	if templateReq.Content != nil {
		updateFields["content"] = *templateReq.Content
	}

	if templateReq.Type != nil {
		updateFields["type"] = *templateReq.Type
	}

	if templateReq.IsPublic != nil {
		updateFields["is_public"] = *templateReq.IsPublic
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No fields to update",
			"message": "没有可更新的字段",
		})
		return
	}

	updateFields["updated_at"] = time.Now()

	err = db.Dao.Model(&template).Where("id = ?", template.ID).Updates(updateFields).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update prompt template",
			"message": "更新提示模板失败",
		})
		return
	}

	db.Dao.Where("id = ?", uint(id)).First(&template)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Prompt template updated successfully",
		"data":    template,
	})
}

// DeletePromptTemplate 删除提示模板
func DeletePromptTemplate(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID format",
			"message": "无效的ID格式",
		})
		return
	}

	var template models.PromptTemplate
	result := db.Dao.Where("id = ? AND user_id = ?", uint(id), userID).First(&template)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Prompt template not found",
			"message": "未找到对应的提示模板",
		})
		return
	}

	if err := db.Dao.Delete(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete prompt template",
			"message": "删除提示模板失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Prompt template deleted successfully",
	})
}
