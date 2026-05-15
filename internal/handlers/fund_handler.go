package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/internal/middleware"
)

// GetFundList 获取基金列表
func GetFundList(c *gin.Context) {
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
	var funds []data.FundBasic

	query := db.Dao.Where("user_id = ?", userID)

	if queryParam := c.Query("q"); queryParam != "" {
		query = query.Where("fund_code LIKE ? OR fund_name LIKE ?", "%"+queryParam+"%", "%"+queryParam+"%")
	}

	if code := c.Query("code"); code != "" {
		query = query.Where("fund_code LIKE ?", "%"+code+"%")
	}

	if name := c.Query("name"); name != "" {
		query = query.Where("fund_name LIKE ?", "%"+name+"%")
	}

	query.Model(&data.FundBasic{}).Count(&total)

	err := query.Offset(offset).Limit(pageSize).Find(&funds).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search funds",
			"message": "搜索基金失败",
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
			"list":       funds,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// FollowFund 关注基金
func FollowFund(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var followReq struct {
		FundCode string `json:"fundCode" binding:"required"`
		FundName string `json:"fundName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&followReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingFollow data.FollowedFund
	result := db.Dao.Where("user_id = ? AND fund_code = ?", userID, followReq.FundCode).First(&existingFollow)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Fund already followed",
			"message": "基金已关注",
		})
		return
	}

	newFollow := &data.FollowedFund{
		UserID:   userID,
		FundCode: followReq.FundCode,
		FundName: followReq.FundName,
	}

	if err := db.Dao.Create(newFollow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to follow fund",
			"message": "关注基金失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Fund followed successfully",
		"data":    newFollow,
	})
}

// UnFollowFund 取消关注基金
func UnFollowFund(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var unfollowReq struct {
		FundCode string `json:"fundCode" binding:"required"`
	}

	if err := c.ShouldBindJSON(&unfollowReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingFollow data.FollowedFund
	result := db.Dao.Where("user_id = ? AND fund_code = ?", userID, unfollowReq.FundCode).First(&existingFollow)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Fund not found in follow list",
			"message": "未关注该基金",
		})
		return
	}

	if err := db.Dao.Delete(&existingFollow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to unfollow fund",
			"message": "取消关注基金失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Fund unfollowed successfully",
	})
}

// GetFundRanking 获取基金排行
func GetFundRanking(c *gin.Context) {
	marketType := c.DefaultQuery("marketType", "kf")
	fundType := c.DefaultQuery("fundType", "all")
	sortField := c.DefaultQuery("sortField", "jnzf")
	sortOrder := c.DefaultQuery("sortOrder", "desc")
	pageIndex, _ := strconv.Atoi(c.DefaultQuery("pageIndex", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	result, err := data.NewFundApi().GetFundRanking(marketType, fundType, sortField, sortOrder, pageIndex, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get fund ranking",
			"message": "获取基金排行失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetFollowedFund 获取关注的基金列表
func GetFollowedFund(c *gin.Context) {
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
	var followedFunds []data.FollowedFund

	query := db.Dao.Where("user_id = ?", userID)

	if code := c.Query("code"); code != "" {
		query = query.Where("fund_code LIKE ?", "%"+code+"%")
	}

	if name := c.Query("name"); name != "" {
		query = query.Where("fund_name LIKE ?", "%"+name+"%")
	}

	query.Model(&data.FollowedFund{}).Count(&total)

	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&followedFunds).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch followed funds",
			"message": "获取关注基金列表失败",
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
			"list":       followedFunds,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}
