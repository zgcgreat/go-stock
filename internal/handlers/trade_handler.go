package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/services"
)

// GetTradeRecords 获取交易记录列表
func GetTradeRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")
	direction := c.Query("direction")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	svc := services.GetTradingService()
	result, err := svc.GetTradingRecordList(data.TradingRecordListQuery{
		Page:      page,
		PageSize:  pageSize,
		Keyword:   keyword,
		Direction: direction,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "获取交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// CreateTradeRecord 创建交易记录
func CreateTradeRecord(c *gin.Context) {
	var record data.TradingRecord

	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if record.TradingTime.IsZero() {
		record.TradingTime = time.Now()
	}

	svc := services.GetTradingService()
	id, err := svc.AddTradingRecord(record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "创建交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    gin.H{"id": id},
	})
}

// UpdateTradeRecord 更新交易记录
func UpdateTradeRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "无效的ID格式",
		})
		return
	}

	var record data.TradingRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	record.ID = uint(id)

	svc := services.GetTradingService()
	if err := svc.UpdateTradingRecord(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "更新交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DeleteTradeRecord 删除交易记录
func DeleteTradeRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "无效的ID格式",
		})
		return
	}

	svc := services.GetTradingService()
	if err := svc.DeleteTradingRecord(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "删除交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}
