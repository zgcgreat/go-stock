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

// GetTradeRecords 获取交易记录列表
func GetTradeRecords(c *gin.Context) {
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
	var records []models.StockTradeRecord

	query := db.Dao.Where("user_id = ?", userID)

	if code := c.Query("code"); code != "" {
		query = query.Where("symbol LIKE ? OR code LIKE ?", "%"+code+"%", "%"+code+"%")
	}

	if tsCode := c.Query("tsCode"); tsCode != "" {
		query = query.Where("ts_code LIKE ?", "%"+tsCode+"%")
	}

	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	if tradeType := c.Query("tradeType"); tradeType != "" {
		query = query.Where("trade_type = ?", tradeType)
	}

	if direction := c.Query("direction"); direction != "" {
		query = query.Where("direction = ?", direction)
	}

	tradingDate := c.Query("tradingDate")
	if tradingDate != "" {
		query = query.Where("DATE(trading_time) = ?", tradingDate)
	}

	query.Model(&models.StockTradeRecord{}).Count(&total)

	err := query.Offset(offset).Limit(pageSize).Order("trading_time DESC, created_at DESC").Find(&records).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch trade records",
			"message": "获取交易记录失败",
		})
		return
	}

	totalPages := total / int64(pageSize)
	if total%int64(pageSize) > 0 {
		totalPages++
	}

	var totalAssets float64
	for _, record := range records {
		if record.Direction == "buy" {
			totalAssets += record.Amount
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":        records,
			"total":       total,
			"page":        page,
			"pageSize":    pageSize,
			"totalPages":  totalPages,
			"totalAssets": totalAssets,
		},
	})
}

// CreateTradeRecord 创建交易记录
func CreateTradeRecord(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	var tradeReq struct {
		TsCode             string    `json:"ts_code" binding:"required"`
		Symbol             string    `json:"symbol" binding:"required"`
		Name               string    `json:"name" binding:"required"`
		Code               string    `json:"code" binding:"required"`
		TradeType          string    `json:"trade_type" binding:"required,oneof=buy sell"`
		Direction          string    `json:"direction" binding:"required,oneof=buy sell"`
		Volume             float64   `json:"volume" binding:"required,gt=0"`
		Price              float64   `json:"price" binding:"required,gt=0"`
		Amount             float64   `json:"amount" binding:"required,gt=0"`
		Commission         float64   `json:"commission,omitempty"`
		Taxes              float64   `json:"taxes,omitempty"`
		TradingTime        time.Time `json:"trading_time"`
		Reason             string    `json:"reason,omitempty"`
		Mindset            string    `json:"mindset,omitempty"`
		ClosePrice         *float64  `json:"close_price,omitempty"`
		CloseVolume        *float64  `json:"close_volume,omitempty"`
		RecordedClosePrice *float64  `json:"recorded_close_price,omitempty"`
	}

	if err := c.ShouldBindJSON(&tradeReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	if tradeReq.TradingTime.IsZero() {
		tradeReq.TradingTime = time.Now()
	}

	newTradeRecord := &models.StockTradeRecord{
		Base:               models.Base{UserID: userID},
		TsCode:             tradeReq.TsCode,
		Symbol:             tradeReq.Symbol,
		Name:               tradeReq.Name,
		Code:               tradeReq.Code,
		TradeType:          tradeReq.TradeType,
		Direction:          tradeReq.Direction,
		Volume:             tradeReq.Volume,
		Price:              tradeReq.Price,
		Amount:             tradeReq.Amount,
		Commission:         tradeReq.Commission,
		Taxes:              tradeReq.Taxes,
		TradingTime:        tradeReq.TradingTime,
		Reason:             tradeReq.Reason,
		Mindset:            tradeReq.Mindset,
		RecordedClosePrice: tradeReq.RecordedClosePrice,
	}

	if tradeReq.ClosePrice != nil && tradeReq.CloseVolume != nil {
		closeTime := time.Now()
		newTradeRecord.CloseTime = &closeTime
		newTradeRecord.ClosePrice = tradeReq.ClosePrice
		newTradeRecord.CloseVolume = tradeReq.CloseVolume
	}

	if err := db.Dao.Create(newTradeRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create trade record",
			"message": "创建交易记录失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "Trade record created successfully",
		"data":    newTradeRecord,
	})
}

// UpdateTradeRecord 更新交易记录
func UpdateTradeRecord(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID format",
			"message": "无效的ID格式",
		})
		return
	}

	var updateReq struct {
		TsCode             *string    `json:"ts_code"`
		Symbol             *string    `json:"symbol"`
		Name               *string    `json:"name"`
		Code               *string    `json:"code"`
		TradeType          *string    `json:"trade_type"`
		Direction          *string    `json:"direction"`
		Volume             *float64   `json:"volume"`
		Price              *float64   `json:"price"`
		Amount             *float64   `json:"amount"`
		Commission         *float64   `json:"commission"`
		Taxes              *float64   `json:"taxes"`
		TradingTime        *time.Time `json:"trading_time"`
		CloseTime          *time.Time `json:"close_time"`
		ClosePrice         *float64   `json:"close_price"`
		CloseVolume        *float64   `json:"close_volume"`
		Reason             *string    `json:"reason"`
		Mindset            *string    `json:"mindset"`
		RecordedClosePrice *float64   `json:"recorded_close_price"`
	}

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var record models.StockTradeRecord
	result := db.Dao.Where("id = ? AND user_id = ?", uint(id), userID).First(&record)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Trade record not found",
			"message": "未找到对应的交易记录",
		})
		return
	}

	updateFields := make(map[string]interface{})

	if updateReq.TsCode != nil {
		updateFields["ts_code"] = *updateReq.TsCode
	}
	if updateReq.Symbol != nil {
		updateFields["symbol"] = *updateReq.Symbol
	}
	if updateReq.Name != nil {
		updateFields["name"] = *updateReq.Name
	}
	if updateReq.Code != nil {
		updateFields["code"] = *updateReq.Code
	}
	if updateReq.TradeType != nil {
		updateFields["trade_type"] = *updateReq.TradeType
	}
	if updateReq.Direction != nil {
		updateFields["direction"] = *updateReq.Direction
	}
	if updateReq.Volume != nil {
		updateFields["volume"] = *updateReq.Volume
	}
	if updateReq.Price != nil {
		updateFields["price"] = *updateReq.Price
	}
	if updateReq.Amount != nil {
		updateFields["amount"] = *updateReq.Amount
	}
	if updateReq.Commission != nil {
		updateFields["commission"] = *updateReq.Commission
	}
	if updateReq.Taxes != nil {
		updateFields["taxes"] = *updateReq.Taxes
	}
	if updateReq.TradingTime != nil {
		updateFields["trading_time"] = *updateReq.TradingTime
	}
	if updateReq.CloseTime != nil {
		updateFields["close_time"] = *updateReq.CloseTime
	}
	if updateReq.ClosePrice != nil {
		updateFields["close_price"] = *updateReq.ClosePrice
	}
	if updateReq.CloseVolume != nil {
		updateFields["close_volume"] = *updateReq.CloseVolume
	}
	if updateReq.Reason != nil {
		updateFields["reason"] = *updateReq.Reason
	}
	if updateReq.Mindset != nil {
		updateFields["mindset"] = *updateReq.Mindset
	}
	if updateReq.RecordedClosePrice != nil {
		updateFields["recorded_close_price"] = *updateReq.RecordedClosePrice
	}

	if len(updateFields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No fields to update",
			"message": "没有可更新的字段",
		})
		return
	}

	updateFields["updated_at"] = time.Now()

	err = db.Dao.Model(&record).Where("id = ?", record.ID).Updates(updateFields).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update trade record",
			"message": "更新交易记录失败",
		})
		return
	}

	db.Dao.Where("id = ?", record.ID).First(&record)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Trade record updated successfully",
		"data":    record,
	})
}

// DeleteTradeRecord 删除交易记录
func DeleteTradeRecord(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User ID not found in context",
			"message": "用户信息异常",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID format",
			"message": "无效的ID格式",
		})
		return
	}

	var record models.StockTradeRecord
	result := db.Dao.Where("id = ? AND user_id = ?", uint(id), userID).First(&record)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Trade record not found",
			"message": "未找到对应的交易记录",
		})
		return
	}

	if err := db.Dao.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete trade record",
			"message": "删除交易记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Trade record deleted successfully",
	})
}
