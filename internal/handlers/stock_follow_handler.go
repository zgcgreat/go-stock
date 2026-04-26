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

func FollowStock(c *gin.Context) {
	// 获取用户ID
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

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

	// 检查当前用户是否已关注
	var existingFollow data.FollowedStock
	result := db.Dao.Where("stock_code = ? AND user_id = ?", followReq.StockCode, userID).First(&existingFollow)
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
		UserID:    userID,
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
		"message": "关注成功",
		"data":    newFollow,
	})
}

func UnfollowStock(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

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

	// 检查当前用户是否关注了该股票
	var existingFollow data.FollowedStock
	result := db.Dao.Where("stock_code = ? AND user_id = ?", stockCode, userID).First(&existingFollow)
	if result.Error != nil {
		fmt.Printf("UnfollowStock: userID=%d, stockCode=%s, error=%v\n", userID, stockCode, result.Error)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not found in follow list",
			"message": "未关注该股票",
		})
		return
	}

	// 只删除当前用户关注的记录
	if err := db.Dao.Where("stock_code = ? AND user_id = ?", stockCode, userID).Delete(&data.FollowedStock{}).Error; err != nil {
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
	// 获取用户ID
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	// 获取groupId参数,默认为0(全部)
	groupId, _ := strconv.Atoi(c.Query("groupId"))

	var followedStocks []data.FollowedStock

	if groupId == 0 {
		// 获取当前用户所有关注的股票
		err := db.Dao.Where("user_id = ?", userID).Model(&data.FollowedStock{}).Order("sort ASC, time DESC").Find(&followedStocks).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch follow list",
				"message": "获取关注列表失败",
			})
			return
		}
	} else {
		// 根据分组ID获取该分组下的股票（仅当前用户的分组）
		groupApi := data.NewStockGroupApi(db.Dao)
		groupStocks := groupApi.GetGroupStockByGroupId(groupId)
		if len(groupStocks) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "success",
				"data":    []data.FollowedStock{},
			})
			return
		}

		// 提取股票代码列表
		stockCodes := make([]string, 0, len(groupStocks))
		for _, gs := range groupStocks {
			stockCodes = append(stockCodes, gs.StockCode)
		}

		// 查询这些股票的详细信息（仅当前用户关注的）
		err := db.Dao.Model(&data.FollowedStock{}).
			Where("stock_code IN ? AND user_id = ?", stockCodes, userID).
			Order("sort ASC, time DESC").
			Find(&followedStocks).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch follow list",
				"message": "获取关注列表失败",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    followedStocks,
	})
}

func SetCostPriceAndVolume(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

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

	// 必须验证股票属于当前用户，防止越权修改
	var follow data.FollowedStock
	result := db.Dao.Where("stock_code = ? AND user_id = ?", costReq.StockCode, userID).First(&follow)
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
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

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

	// 必须验证股票属于当前用户，防止越权修改
	var follow data.FollowedStock
	result := db.Dao.Where("stock_code = ? AND user_id = ?", alarmReq.StockCode, userID).First(&follow)
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
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

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

	// 必须验证股票属于当前用户，防止越权修改
	result := db.Dao.Model(&data.FollowedStock{}).
		Where("stock_code = ? AND user_id = ?", sortReq.StockCode, userID).
		Update("sort", sortReq.Sort)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not followed",
			"message": "请先关注该股票",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Stock sort order updated successfully",
	})
}
