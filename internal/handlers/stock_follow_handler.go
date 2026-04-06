package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
)

func FollowStock(c *gin.Context) {
	var followReq struct {
		StockCode string `json:"stockCode" binding:"required"`
		Name      string `json:"name"`
	}

	if err := c.ShouldBindJSON(&followReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingFollow data.FollowedStock
	result := db.Dao.Where("stock_code = ?", followReq.StockCode).First(&existingFollow)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Stock already followed",
			"message": "股票已关注",
		})
		return
	}

	newFollow := &data.FollowedStock{
		StockCode: followReq.StockCode,
		Name:      followReq.Name,
	}

	if err := db.Dao.Create(newFollow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to follow stock",
			"message": "关注股票失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Stock followed successfully",
		"data":    newFollow,
	})
}

func UnfollowStock(c *gin.Context) {
	var stockCode string

	// First, try getting stockCode from JSON body
	var unfollowReq struct {
		StockCode string `json:"stockCode"`
	}

	jsonErr := c.ShouldBindJSON(&unfollowReq)
	if jsonErr == nil && unfollowReq.StockCode != "" {
		stockCode = unfollowReq.StockCode
	} else {
		// Fallback to getting stockCode from URL path parameter
		stockCode = c.Param("code")
		if stockCode == "" {
			// Also check if it's sent as a form/query parameter
			stockCode = c.Query("stockCode")
		}
	}

	if stockCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing stock code",
			"message": "股票代码不能为空",
		})
		return
	}

	var existingFollow data.FollowedStock
	result := db.Dao.Unscoped().Where("stock_code = ?", stockCode).First(&existingFollow)
	if result.Error != nil {
		fmt.Printf("UnfollowStock: stockCode=%s, error=%v\n", stockCode, result.Error)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not found in follow list",
			"message": "未关注该股票",
		})
		return
	}

	if err := db.Dao.Unscoped().Where("stock_code = ?", stockCode).Delete(&data.FollowedStock{}).Error; err != nil {
		fmt.Printf("UnfollowStock delete error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to unfollow stock",
			"message": "取消关注股票失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "取消关注成功",
	})
}

func GetFollowList(c *gin.Context) {
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
	var followedStocks []data.FollowedStock

	query := db.Dao.Model(&data.FollowedStock{})

	if code := c.Query("code"); code != "" {
		query = query.Where("stock_code LIKE ?", "%"+code+"%")
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	query.Count(&total)
	err := query.Offset(offset).Limit(pageSize).Order("sort ASC").Find(&followedStocks).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch follow list",
			"message": "获取关注列表失败",
		})
		return
	}

	totalPages := int(total) / pageSize
	if total%int64(pageSize) > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       followedStocks,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

func SetCostPriceAndVolume(c *gin.Context) {
	var costReq struct {
		StockCode string  `json:"stockCode" binding:"required"`
		CostPrice float64 `json:"costPrice"`
		Volume    int64   `json:"volume"`
	}

	if err := c.ShouldBindJSON(&costReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var follow data.FollowedStock
	result := db.Dao.Where("stock_code = ?", costReq.StockCode).First(&follow)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not followed",
			"message": "请先关注该股票",
		})
		return
	}

	db.Dao.Model(&follow).Updates(map[string]interface{}{
		"cost_price": costReq.CostPrice,
		"volume":     costReq.Volume,
	})

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Cost info updated successfully",
	})
}

func SetAlarmChangePercent(c *gin.Context) {
	var alarmReq struct {
		StockCode          string  `json:"stockCode" binding:"required"`
		AlarmChangePercent float64 `json:"alarmChangePercent"`
	}

	if err := c.ShouldBindJSON(&alarmReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var follow data.FollowedStock
	result := db.Dao.Where("stock_code = ?", alarmReq.StockCode).First(&follow)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not followed",
			"message": "请先关注该股票",
		})
		return
	}

	db.Dao.Model(&follow).Update("alarm_change_percent", alarmReq.AlarmChangePercent)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Alarm setting updated successfully",
	})
}

func SetStockSort(c *gin.Context) {
	var sortReq struct {
		StockCode string `json:"stockCode" binding:"required"`
		Sort      int64  `json:"sort"`
	}

	if err := c.ShouldBindJSON(&sortReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	db.Dao.Model(&data.FollowedStock{}).
		Where("stock_code = ?", sortReq.StockCode).
		Update("sort", sortReq.Sort)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Stock sort order updated successfully",
	})
}
