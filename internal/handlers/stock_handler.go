package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/internal/middleware"
)

// GetStockBasics 获取股票基本信息
func GetStockBasics(c *gin.Context) {
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
	var stockBasics []data.StockBasic

	query := db.Dao.Model(&data.StockBasic{})

	// 添加查询参数过滤
	if code := c.Query("code"); code != "" {
		query = query.Where("ts_code LIKE ? OR symbol LIKE ?", "%"+code+"%", "%"+code+"%")
	}
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if industry := c.Query("industry"); industry != "" {
		query = query.Where("industry LIKE ?", "%"+industry+"%")
	}
	if market := c.Query("market"); market != "" {
		query = query.Where("market LIKE ?", "%"+market+"%")
	}

	query.Count(&total)
	err := query.Offset(offset).Limit(pageSize).Find(&stockBasics).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch stock basics",
			"message": "获取股票列表失败",
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
			"list":       stockBasics,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// GetStockByCode 根据股票代码获取股票信息
func GetStockByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Stock code is required",
			"message": "股票代码不能为空",
		})
		return
	}

	var stockBasic data.StockBasic
	result := db.Dao.Where("ts_code = ? OR symbol = ?", code, code).First(&stockBasic)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Stock not found",
			"message": "未找到该股票",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stockBasic,
	})
}

// GetStockKLine 获取股票K线数据
func GetStockKLine(c *gin.Context) {
	stockCode := c.Param("code")
	if stockCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Stock code is required",
			"message": "股票代码不能为空",
		})
		return
	}

	kLineType := c.DefaultQuery("klt", "101") // 默认日K
	// 同时支持 days 和 limit 参数（前端可能用 limit）
	days, _ := strconv.Atoi(c.DefaultQuery("days", ""))
	if days <= 0 {
		days, _ = strconv.Atoi(c.DefaultQuery("limit", "120"))
	}
	if days <= 0 {
		days = 120
	}

	// 复用桌面端逻辑：优先东方财富，失败时自动切换其他数据源
	result := data.FetchKLineWithFallback(stockCode, "", kLineType, days, "")
	kLines := &[]data.KLineData{}
	if result != nil && result.Data != nil {
		kLines = result.Data
	}

	// Web端返回格式与桌面端保持一致：直接返回数组
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    kLines,
	})
}

// GetStockRealTime 获取股票实时行情
func GetStockRealTime(c *gin.Context) {
	codes := c.Query("codes")
	if codes == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Stock codes are required",
			"message": "股票代码不能为空",
		})
		return
	}

	// 获取用户ID（匿名用户 userID=0，只返回行情不填充持仓信息）
	userID, _ := middleware.GetUserIDFromContext(c)

	stockApi := data.NewStockDataApi()
	stocks, _ := stockApi.GetStockCodeRealTimeData(codes)

	// 为每只股票填充关注信息和计算字段（仅限当前用户的持仓数据）
	for i := range *stocks {
		follow := &data.FollowedStock{
			StockCode: (*stocks)[i].Code,
		}
		if userID > 0 {
			// 有登录用户：只查当前用户的持仓信息
			db.Dao.Model(follow).Where("stock_code = ? AND user_id = ?", (*stocks)[i].Code, userID).First(follow)
		}
		// 调用 addStockFollowData 计算涨跌幅、盈亏等字段
		addStockFollowData(*follow, &(*stocks)[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stocks,
	})
}

// SearchStocks 搜索股票
// 直接复用桌面端 data.StockDataApi.GetStockList 的逻辑，确保 Web 端与桌面端行为一致：
//   - 覆盖 StockBasic / IndexBasic / StockInfoHK / StockInfoUS / AllStockInfo 五张表
//   - 有去重处理
//   - keyword 为空时返回空列表（前端预加载时由 wails-bridge 短路，不到达此处）
func SearchStocks(c *gin.Context) {
	keyword := c.Query("keyword")
	// 允许 keyword 为空，直接复用桌面端逻辑（空关键词返回空列表）
	results := data.NewStockDataApi().GetStockList(keyword)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    results,
	})
}

// GetStockRealTimePrice 获取股票实时价格数据
func GetStockRealTimePrice(c *gin.Context) {
	codes := c.Query("codes")
	if codes == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Stock codes are required",
			"message": "股票代码不能为空",
		})
		return
	}

	stockApi := data.NewStockDataApi()
	results := make(map[string]interface{})

	// 解析多个股票代码
	codeList := strings.Split(codes, ",")
	for _, code := range codeList {
		code = strings.TrimSpace(code)
		if code != "" {
			minuteData, date := stockApi.GetStockMinutePriceData(code)
			results[code] = gin.H{
				"minuteData": minuteData,
				"date":       date,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    results,
	})
}

// addStockFollowData 已提取到 data.AddStockFollowData，此处为薄转发
func addStockFollowData(follow data.FollowedStock, stockData *data.StockInfo) {
	data.AddStockFollowData(follow, stockData)
}
