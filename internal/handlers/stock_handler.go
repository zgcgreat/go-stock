package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/mathutil"
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

	// Web端返回格式与桌面端保持一致：直接返回数组
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"kline": kLines,
		},
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

	// 为每只股票填充关注信息和计算字段（与桌面端保持一致）
	for i := range *stocks {
		follow := &data.FollowedStock{
			StockCode: (*stocks)[i].Code,
		}
		db.Dao.Model(follow).Where("stock_code = ?", (*stocks)[i].Code).First(follow)
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

// addStockFollowData 为股票数据添加关注信息和计算字段（与桌面端app.go中的逻辑一致）
func addStockFollowData(follow data.FollowedStock, stockData *data.StockInfo) {
	stockData.PrePrice = follow.Price //上次当前价格
	stockData.Sort = follow.Sort
	stockData.CostPrice = follow.CostPrice //成本价
	stockData.CostVolume = follow.Volume   //成本量
	stockData.AlarmChangePercent = follow.AlarmChangePercent
	stockData.AlarmPrice = follow.AlarmPrice

	//当前价格
	price, _ := convertor.ToFloat(stockData.Price)
	//当前价格为0 时 使用卖一价格作为当前价格
	if price == 0 {
		price, _ = convertor.ToFloat(stockData.A1P)
	}
	//当前价格依然为0 时 使用买一报价作为当前价格
	if price == 0 {
		price, _ = convertor.ToFloat(stockData.B1P)
	}

	//昨日收盘价
	preClosePrice, _ := convertor.ToFloat(stockData.PreClose)

	//当前价格依然为0 时 使用昨日收盘价为当前价格
	if price == 0 {
		price = preClosePrice
	}

	//今日最高价
	highPrice, _ := convertor.ToFloat(stockData.High)
	if highPrice == 0 {
		highPrice, _ = convertor.ToFloat(stockData.Open)
	}

	//今日最低价
	lowPrice, _ := convertor.ToFloat(stockData.Low)
	if lowPrice == 0 {
		lowPrice, _ = convertor.ToFloat(stockData.Open)
	}

	if price > 0 && preClosePrice > 0 {
		stockData.ChangePrice = mathutil.RoundToFloat(price-preClosePrice, 2)
		stockData.ChangePercent = mathutil.RoundToFloat(mathutil.Div(price-preClosePrice, preClosePrice)*100, 3)
	}
	if highPrice > 0 && preClosePrice > 0 {
		stockData.HighRate = mathutil.RoundToFloat(mathutil.Div(highPrice-preClosePrice, preClosePrice)*100, 3)
	}
	if lowPrice > 0 && preClosePrice > 0 {
		stockData.LowRate = mathutil.RoundToFloat(mathutil.Div(lowPrice-preClosePrice, preClosePrice)*100, 3)
	}
	if follow.CostPrice > 0 && follow.Volume > 0 {
		if price > 0 {
			stockData.Profit = mathutil.RoundToFloat(mathutil.Div(price-follow.CostPrice, follow.CostPrice)*100, 3)
			stockData.ProfitAmount = mathutil.RoundToFloat((price-follow.CostPrice)*float64(follow.Volume), 2)
			stockData.ProfitAmountToday = mathutil.RoundToFloat((price-preClosePrice)*float64(follow.Volume), 2)
		} else {
			//未开盘时当前价格为昨日收盘价
			stockData.Profit = mathutil.RoundToFloat(mathutil.Div(preClosePrice-follow.CostPrice, follow.CostPrice)*100, 3)
			stockData.ProfitAmount = mathutil.RoundToFloat((preClosePrice-follow.CostPrice)*float64(follow.Volume), 2)
			// 未开盘时，今日盈亏为 0
			stockData.ProfitAmountToday = 0
		}
	}
}
