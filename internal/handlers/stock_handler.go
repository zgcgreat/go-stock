package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"
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
	days, _ := strconv.Atoi(c.DefaultQuery("days", "120"))
	adjustFlag := c.DefaultQuery("fqt", "") // qfq=前复权, hfq=后复权

	config := data.GetSettingConfig()
	if config == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Config not found",
			"message": "配置获取失败",
		})
		return
	}

	api := data.NewEastMoneyKLineApi(config)
	kLines := api.GetKLineDataBefore(stockCode, kLineType, adjustFlag, days, "")

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

	stockApi := data.NewStockDataApi()
	stocks, _ := stockApi.GetStockCodeRealTimeData(codes)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stocks,
	})
}

// SearchStocks 搜索股票
func SearchStocks(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Keyword is required",
			"message": "搜索关键词不能为空",
		})
		return
	}

	var results []data.StockBasic
	db.Dao.Where("name LIKE ? OR ts_code LIKE ? OR symbol LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Limit(50).Find(&results)

	// 搜索全部股票信息表
	var allStockResults []models.AllStockInfo
	db.Dao.Model(&models.AllStockInfo{}).Where("SECURITYNAMEABBR LIKE ? OR SECUCODE LIKE ? OR SECURITYCODE LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Limit(50).Find(&allStockResults)
	for _, item := range allStockResults {
		market := item.MARKET
		if market == "上海证券交易所" || market == "SZSE" {
			market = "SSE"
		} else if market == "深圳证券交易所" || market == "SSE" {
			market = "SZSE"
		}
		tsCode := item.SECUCODE
		if tsCode == "" {
			tsCode = item.SECURITYCODE
		}
		results = append(results, data.StockBasic{
			TsCode:   tsCode,
			Name:     item.SECURITYNAMEABBR,
			Symbol:   item.SECURITYCODE,
			Market:   market,
			Industry: item.INDUSTRY,
		})
	}

	// 搜索指数
	var results2 []data.IndexBasic
	db.Dao.Where("market IN ?", []string{"SSE", "SZSE"}).Where("name LIKE ? OR ts_code LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(20).Find(&results2)
	for _, item := range results2 {
		results = append(results, data.StockBasic{
			TsCode:   item.TsCode,
			Name:     item.Name,
			Fullname: item.FullName,
			Symbol:   item.Symbol,
			Market:   item.Market,
		})
	}

	// 搜索港股
	var results3 []models.StockInfoHK
	db.Dao.Model(&models.StockInfoHK{}).Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(20).Find(&results3)
	for _, item := range results3 {
		results = append(results, data.StockBasic{
			TsCode: "hk" + item.Code,
			Name:   item.Name,
			Symbol: item.Code,
			Market: "HK",
		})
	}

	// 搜索美股
	var results4 []models.StockInfoUS
	db.Dao.Model(&models.StockInfoUS{}).Where("name LIKE ? OR code LIKE ? OR e_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Limit(20).Find(&results4)
	for _, item := range results4 {
		results = append(results, data.StockBasic{
			TsCode: "us" + item.Code,
			Name:   item.Name,
			Symbol: item.Code,
			Market: "US",
		})
	}

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
