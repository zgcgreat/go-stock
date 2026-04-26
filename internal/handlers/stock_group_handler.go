package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/internal/middleware"
)

// GetGroupList 获取分组列表
func GetGroupList(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	// DEBUG: 打印当前用户ID
	fmt.Printf("[DEBUG] GetGroupList userID=%d\n", userID)

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
	var groups []data.Group

	query := db.Dao.Where("user_id = ?", userID)

	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	query.Model(&data.Group{}).Count(&total)

	err := query.Offset(offset).Limit(pageSize).Order("sort ASC, created_at DESC").Find(&groups).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch group list",
			"message": "获取分组列表失败",
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
			"list":       groups,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// AddGroup 添加分组
func AddGroup(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var groupReq struct {
		Name string `json:"name" binding:"required"`
		Sort int    `json:"sort,omitempty"`
		Desc string `json:"desc,omitempty"`
	}

	if err := c.ShouldBindJSON(&groupReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingGroup data.Group
	result := db.Dao.Where("user_id = ? AND name = ?", userID, groupReq.Name).First(&existingGroup)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Group name already exists",
			"message": "分组名称已存在",
		})
		return
	}

	if groupReq.Sort == 0 {
		var maxGroup data.Group
		db.Dao.Where("user_id = ?", userID).Order("sort DESC").First(&maxGroup)
		groupReq.Sort = maxGroup.Sort + 1
	}

	newGroup := &data.Group{
		UserID: userID,
		Name:   groupReq.Name,
		Sort:   groupReq.Sort,
		Desc:   groupReq.Desc,
	}

	if err := db.Dao.Create(newGroup).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create group",
			"message": "创建分组失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Group added successfully",
		"data":    newGroup,
	})
}

// UpdateGroupSort 更新分组排序
func UpdateGroupSort(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var sortReq struct {
		Groups []struct {
			ID   uint `json:"id" binding:"required"`
			Sort int  `json:"sort" binding:"required"`
		} `json:"groups" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&sortReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	tx := db.Dao.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, groupData := range sortReq.Groups {
		var group data.Group
		result := tx.Where("id = ? AND user_id = ?", groupData.ID, userID).First(&group)
		if result.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Group not found",
				"message": "分组不存在或不属于当前用户",
			})
			return
		}

		if err := tx.Model(&group).Update("sort", groupData.Sort).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update group sort",
				"message": "更新分组排序失败",
			})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to commit transaction",
			"message": "提交更改失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Group sort updated successfully",
	})
}

// GetGroupStockList 获取分组股票列表
func GetGroupStockList(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	groupIDStr := c.Param("groupId")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid group ID",
			"message": "无效的分组ID",
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
	var groupStocks []data.GroupStock

	query := db.Dao.Where("group_id = ? AND user_id = ?", uint(groupID), userID)

	if stockCode := c.Query("stockCode"); stockCode != "" {
		query = query.Where("stock_code LIKE ?", "%"+stockCode+"%")
	}

	if stockName := c.Query("stockName"); stockName != "" {
		query = query.Where("stock_name LIKE ?", "%"+stockName+"%")
	}

	query.Model(&data.GroupStock{}).Count(&total)

	err = query.Offset(offset).Limit(pageSize).Order("sort ASC, created_at DESC").Find(&groupStocks).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch group stock list",
			"message": "获取分组股票列表失败",
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
			"list":       groupStocks,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// AddStockGroup 添加股票到分组
func AddStockGroup(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var stockGroupReq struct {
		GroupID   uint   `json:"groupId" binding:"required"`
		StockCode string `json:"stockCode" binding:"required"`
		StockName string `json:"stockName" binding:"required"`
		Sort      int    `json:"sort,omitempty"`
	}

	if err := c.ShouldBindJSON(&stockGroupReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var group data.Group
	result := db.Dao.Where("id = ? AND user_id = ?", stockGroupReq.GroupID, userID).First(&group)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Group not found",
			"message": "分组不存在或不属于当前用户",
		})
		return
	}

	var existingGroupStock data.GroupStock
	result = db.Dao.Where("group_id = ? AND stock_code = ? AND user_id = ?",
		stockGroupReq.GroupID, stockGroupReq.StockCode, userID).First(&existingGroupStock)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Stock already in group",
			"message": "股票已在分组中",
		})
		return
	}

	if stockGroupReq.Sort == 0 {
		var maxGroupStock data.GroupStock
		db.Dao.Where("group_id = ?", stockGroupReq.GroupID).Order("sort DESC").First(&maxGroupStock)
		stockGroupReq.Sort = maxGroupStock.Sort + 1
	}

	newGroupStock := &data.GroupStock{
		UserID:    userID,
		GroupID:   stockGroupReq.GroupID,
		StockCode: stockGroupReq.StockCode,
		StockName: stockGroupReq.StockName,
		Sort:      stockGroupReq.Sort,
	}

	if err := db.Dao.Create(newGroupStock).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add stock to group",
			"message": "添加股票到分组失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Stock added to group successfully",
		"data":    newGroupStock,
	})
}
