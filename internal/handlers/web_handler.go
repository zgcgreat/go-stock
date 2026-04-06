package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"
	"go-stock/backend/services"
)

// GetTelegraphList 获取新闻电报列表
func GetTelegraphList(c *gin.Context) {
	source := c.DefaultQuery("source", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	news := data.NewMarketNewsApi().GetTelegraphList(source)
	if limit > 0 && len(*news) > limit {
		*news = (*news)[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": news})
}

// RefreshTelegraphList 刷新新闻电报
func RefreshTelegraphList(c *gin.Context) {
	source := c.DefaultQuery("source", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	switch source {
	case "财联社电报":
		data.NewMarketNewsApi().TelegraphList(30)
	case "新浪财经":
		data.NewMarketNewsApi().GetSinaNews(30)
	}
	news := data.NewMarketNewsApi().GetTelegraphList(source)
	if limit > 0 && len(*news) > limit {
		*news = (*news)[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": news})
}

// GlobalStockIndexes 获取全球指数
func GlobalStockIndexes(c *gin.Context) {
	// 先尝试从缓存获取
	indexes := data.NewMarketNewsApi().GetCachedGlobalStockIndexes("all")
	if indexes != nil && len(*indexes) > 0 {
		// 按区域组织
		result := map[string][]map[string]any{}
		for _, idx := range *indexes {
			result[idx.Region] = append(result[idx.Region], map[string]any{
				"code":     idx.Code,
				"name":     idx.Name,
				"location": idx.Location,
				"qtcode":   idx.Qtcode,
				"state":    idx.State,
				"zdf":      idx.Zdf,
				"zxj":      idx.Zxj,
				"img":      idx.Img,
			})
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
		return
	}

	// 缓存没有则从接口获取并缓存
	go func() {
		data.NewMarketNewsApi().CacheGlobalStockIndexes(30)
	}()

	// 直接从API获取
	apiData := data.NewMarketNewsApi().GlobalStockIndexes(30)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": apiData})
}

// GetIndustryRank 获取行业排名
func GetIndustryRank(c *gin.Context) {
	sort := c.DefaultQuery("sort", "0")
	cnt, _ := strconv.Atoi(c.DefaultQuery("cnt", "150"))
	res := data.NewMarketNewsApi().GetIndustryRank(sort, cnt)
	// 腾讯API返回 {"code":0,"msg":"ok","data":[{bd_name,...},...]}
	// data 直接就是数组
	if res != nil {
		if items, ok := res["data"].([]any); ok {
			c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
			return
		}
		// 兼容旧格式：data 可能是 map{"item":[...]}
		if dataMap, ok := res["data"].(map[string]any); ok {
			if items, exists := dataMap["item"]; exists {
				c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
				return
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []any{}})
}

// GetIndustryMoneyRankSina 获取行业资金排名
func GetIndustryMoneyRankSina(c *gin.Context) {
	fenlei := c.DefaultQuery("fenlei", "0")
	sort := c.DefaultQuery("sort", "netamount")
	res := data.NewMarketNewsApi().GetIndustryMoneyRankSina(fenlei, sort)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetMoneyRankSina 获取个股资金排名
func GetMoneyRankSina(c *gin.Context) {
	sort := c.DefaultQuery("sort", "netamount")
	res := data.NewMarketNewsApi().GetMoneyRankSina(sort)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetStockMoneyTrendByDay 获取个股资金趋势
func GetStockMoneyTrendByDay(c *gin.Context) {
	stockCode := c.Param("code")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "10"))
	res := data.NewMarketNewsApi().GetStockMoneyTrendByDay(stockCode, days)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetLongTigerRank 获取龙虎榜
func GetLongTigerRank(c *gin.Context) {
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	ranks := data.NewMarketNewsApi().LongTiger(date)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ranks})
}

// GetStockResearchReport 获取个股研报
func GetStockResearchReport(c *gin.Context) {
	stockCode := c.DefaultQuery("stockCode", "")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "90"))
	res := data.NewMarketNewsApi().StockResearchReport(stockCode, days)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetStockNotice 获取公司公告
func GetStockNotice(c *gin.Context) {
	stockCode := c.DefaultQuery("stockCode", "")
	res := data.NewMarketNewsApi().StockNotice(stockCode)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetIndustryResearchReport 获取行业研究报告
func GetIndustryResearchReport(c *gin.Context) {
	industryCode := c.DefaultQuery("industryCode", "")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "90"))
	res := data.NewMarketNewsApi().IndustryResearchReport(industryCode, days)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetHotStock 获取热门股票
func GetHotStock(c *gin.Context) {
	marketType := c.DefaultQuery("marketType", "10")
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	res := data.NewMarketNewsApi().XUEQIUHotStock(size, marketType)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetHotEvent 获取热门事件
func GetHotEvent(c *gin.Context) {
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	res := data.NewMarketNewsApi().HotEvent(size)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetHotTopic 获取热门话题
func GetHotTopic(c *gin.Context) {
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	res := data.NewMarketNewsApi().HotTopic(size)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetInvestCalendar 获取投资日历
func GetInvestCalendar(c *gin.Context) {
	yearMonth := c.DefaultQuery("yearMonth", "")
	res := data.NewMarketNewsApi().InvestCalendar(yearMonth)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetClsCalendar 获取财经日历
func GetClsCalendar(c *gin.Context) {
	res := data.NewMarketNewsApi().ClsCalendar()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetEMDictCode 获取东方财富板块代码
func GetEMDictCode(c *gin.Context) {
	code := c.Param("code")
	cache := data.NewMarketNewsApi()
	_ = cache
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": code})
}

// GetTradingRecordStatistics 获取交易记录统计
func GetTradingRecordStatistics(c *gin.Context) {
	svc := services.GetTradingService()
	stats, err := svc.GetTradingRecordStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "获取统计数据失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// GetAllStockInfoList 获取所有股票信息列表
func GetAllStockInfoList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	keyword := c.DefaultQuery("keyword", "")

	var total int64
	var stocks []models.AllStockInfo

	query := db.Dao.Model(&models.AllStockInfo{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	query.Count(&total)

	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Find(&stocks)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":     stocks,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// GetSponsorInfo 获取赞助信息
func GetSponsorInfo(c *gin.Context) {
	level, active := data.EffectiveSponsorVipLevel()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"sponsor":  active,
			"vipLevel": level,
			"active":   active,
		},
	})
}

// GetCronTaskTypes 获取定时任务类型
func GetCronTaskTypes(c *gin.Context) {
	types := []map[string]string{
		{"name": "ai_analyze", "label": "AI分析"},
		{"name": "data_sync", "label": "数据同步"},
		{"name": "news_summary", "label": "新闻摘要"},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": types})
}

// ShareText 分享文本
func ShareText(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": "分享成功"})
}

// ShareAnalysis 分享分析
func ShareAnalysis(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": "分享成功"})
}

// SendDingDingMessage 发送钉钉消息
func SendDingDingMessage(c *gin.Context) {
	var req struct {
		Message   string `json:"message"`
		StockCode string `json:"stockCode"`
		Title     string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	title := req.Title
	if title == "" {
		title = req.StockCode
	}
	msg := req.Message
	if msg == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": "消息为空，跳过发送"})
		return
	}
	result := data.NewDingDingAPI().SendToDingDing(title, msg)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// CheckFrequentTrading 检查频繁交易
func CheckFrequentTrading(c *gin.Context) {
	stockCode := c.Query("stockCode")
	if stockCode == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": false})
		return
	}
	result, _ := data.NewStockDataApi().CheckFrequentTrading(stockCode)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// FetchAiModels 获取AI模型列表
func FetchAiModels(c *gin.Context) {
	baseUrl := strings.TrimSpace(c.Query("baseUrl"))
	apiKey := strings.TrimSpace(c.Query("apiKey"))

	if baseUrl == "" || apiKey == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []string{}})
		return
	}

	type modelItem struct {
		ID string `json:"id"`
	}
	var respData struct {
		Data []modelItem `json:"data"`
	}

	client := resty.New()
	client.SetBaseURL(baseUrl)
	client.SetHeader("Authorization", "Bearer "+apiKey)
	client.SetHeader("Content-Type", "application/json")

	resp, err := client.R().SetResult(&respData).Get("/models")
	if err != nil {
		logger.SugaredLogger.Errorf("FetchAiModels error: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []string{}})
		return
	}
	if resp.IsError() {
		logger.SugaredLogger.Errorf("FetchAiModels http error: %s", resp.Status())
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []string{}})
		return
	}

	modelsList := make([]string, 0, len(respData.Data))
	for _, m := range respData.Data {
		if strings.TrimSpace(m.ID) != "" {
			modelsList = append(modelsList, m.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": modelsList})
}

// GetHotStrategy 获取热门选股策略（从东方财富接口）
func GetHotStrategy(c *gin.Context) {
	res := data.NewSearchStockApi("").HotStrategy()
	c.JSON(http.StatusOK, res)
}

// IndicatorSearchStock 指标选股（从东方财富接口）
func IndicatorSearchStock(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "搜索关键词不能为空",
		})
		return
	}
	res := data.NewSearchStockApi(keyword).SearchStock(5000)
	c.JSON(http.StatusOK, res)
}

// AnalyzeSentiment 分析情感
func AnalyzeSentiment(c *gin.Context) {
	text := c.DefaultQuery("text", "")
	result := data.AnalyzeSentiment(text)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"score":       result.Score,
			"category":    result.Category,
			"description": result.Description,
		},
	})
}

// GetStockChanges 获取股票异动数据（实时）
func GetStockChanges(c *gin.Context) {
	changeTypesStr := c.Query("changeTypes")
	var changeTypes []int
	if changeTypesStr != "" {
		for _, s := range strings.Split(changeTypesStr, ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				changeTypes = append(changeTypes, v)
			}
		}
	}

	pageIndex, _ := strconv.Atoi(c.DefaultQuery("pageIndex", "0"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	result := data.NewStockChangesApi().GetStockChanges(changeTypes, pageIndex, pageSize)
	if result == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":       0,
			"message":    "success",
			"data":       []any{},
			"totalCount": 0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"message":    "success",
		"data":       result.Data,
		"totalCount": result.TotalCount,
	})
}

// GetAllStockChangesWithPaging 获取全部异动数据（带分页）
func GetAllStockChangesWithPagingHandler(c *gin.Context) {
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "500"))

	result := data.NewStockChangesApi().GetAllStockChangesWithPaging(pageSize)
	if result == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":       0,
			"message":    "success",
			"data":       []any{},
			"totalCount": 0,
		})
		return
	}

	// 保存到历史
	historyService := data.NewStockChangeHistoryService()
	_, _ = historyService.SaveStockChangesWithDedup(result.Data)

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"message":    "success",
		"data":       result.Data,
		"totalCount": result.TotalCount,
	})
}

// SaveStockChangesToHistoryHandler 保存异动数据到历史
func SaveStockChangesToHistoryHandler(c *gin.Context) {
	changeTypesStr := c.Query("changeTypes")
	var changeTypes []int
	if changeTypesStr != "" {
		for _, s := range strings.Split(changeTypesStr, ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				changeTypes = append(changeTypes, v)
			}
		}
	}

	result := data.NewStockChangesApi().GetStockChanges(changeTypes, 0, 500)
	if result == nil || len(result.Data) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "没有获取到异动数据",
		})
		return
	}

	err := data.NewStockChangeHistoryService().SaveStockChanges(result.Data)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "保存失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("成功保存 %d 条异动数据", len(result.Data)),
	})
}

// GetStockChangeHistoryHandler 获取异动历史数据
func GetStockChangeHistoryHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	stockCode := c.Query("stockCode")
	stockName := c.Query("stockName")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	changeTypesStr := c.Query("changeTypes")
	var changeTypes []int
	if changeTypesStr != "" {
		for _, s := range strings.Split(changeTypesStr, ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				changeTypes = append(changeTypes, v)
			}
		}
	}

	query := models.StockChangeHistoryQuery{
		Page:        page,
		PageSize:    pageSize,
		StockCode:   stockCode,
		StockName:   stockName,
		StartDate:   startDate,
		EndDate:     endDate,
		ChangeTypes: changeTypes,
	}

	result, err := data.NewStockChangeHistoryService().GetHistoryList(query)
	if err != nil || result == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"list":       []any{},
				"total":      0,
				"page":       page,
				"pageSize":   pageSize,
				"totalPages": 0,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       result.List,
			"total":      result.Total,
			"page":       result.Page,
			"pageSize":   result.PageSize,
			"totalPages": result.TotalPages,
		},
	})
}
