package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"

	"go-stock/backend/agent"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"
	"go-stock/backend/services"
	"go-stock/internal/middleware"
)

// CronSchedulerHooks 定时任务调度器回调（由 webserver 注册，避免循环导入）
var CronSchedulerHooks struct {
	AddCronTask    func(task *models.CronTask) error
	RemoveCronTask func(taskID uint)
}

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
	case "外媒":
		data.NewMarketNewsApi().TradingViewNews()
	}
	news := data.NewMarketNewsApi().GetTelegraphList(source)
	if limit > 0 && len(*news) > limit {
		*news = (*news)[:limit]
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": news})
}

// GlobalStockIndexes 获取全球指数
func GlobalStockIndexes(c *gin.Context) {
	// 直接从API获取最新数据
	apiData := data.NewMarketNewsApi().GlobalStockIndexes(30)
	if apiData == nil {
		apiData = map[string]any{}
	}

	// 返回与桌面端一致的格式
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"msg":     "success",
		"data":    apiData,
		"message": "success",
	})
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

// GetTradingRecordStatistics 获取交易记录统计（用户隔离）
func GetTradingRecordStatistics(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	svc := services.GetTradingService()
	stats, err := svc.GetTradingRecordStatistics(userID)
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

// GetAllStocks 获取股票列表（技术面筛选）
func GetAllStocks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	keyword := c.DefaultQuery("keyword", "")

	// 解析技术指标筛选条件
	indicators := models.TechnicalIndicators{}
	// 从请求参数解析布尔类型指标
	if c.Query("MACD_GOLDEN_FORK") == "true" {
		indicators.MACDGOLDENFORK = true
	}
	if c.Query("KDJ_GOLDEN_FORK") == "true" {
		indicators.KDJGOLDENFORK = true
	}
	if c.Query("BREAK_THROUGH") == "true" {
		indicators.BREAKTHROUGH = true
	}
	if c.Query("LOW_FUNDS_INFLOW") == "true" {
		indicators.LOWFUNDSINFLOW = true
	}
	if c.Query("HIGH_FUNDS_OUTFLOW") == "true" {
		indicators.HIGHFUNDSOUTFLOW = true
	}
	if c.Query("BREAKUP_MA_5DAYS") == "true" {
		indicators.BREAKUPMA5DAYS = true
	}
	if c.Query("LONG_AVG_ARRAY") == "true" {
		indicators.LONGAVGARRAY = true
	}
	if c.Query("SHORT_AVG_ARRAY") == "true" {
		indicators.SHORTAVGARRAY = true
	}
	if c.Query("UPPER_LARGE_VOLUME") == "true" {
		indicators.UPPERLARGEVOLUME = true
	}
	if c.Query("DOWN_NARROW_VOLUME") == "true" {
		indicators.DOWNNARROWVOLUME = true
	}
	if c.Query("ONE_DAYANG_LINE") == "true" {
		indicators.ONEDAYANGLINE = true
	}
	if c.Query("TWO_DAYANG_LINES") == "true" {
		indicators.TWODAYANGLINES = true
	}
	if c.Query("RISE_SUN") == "true" {
		indicators.RISESUN = true
	}
	if c.Query("POWER_FULGUN") == "true" {
		indicators.POWERFULGUN = true
	}
	if c.Query("RESTORE_JUSTICE") == "true" {
		indicators.RESTOREJUSTICE = true
	}
	if c.Query("DOWN_7DAYS") == "true" {
		indicators.DOWN7DAYS = true
	}
	if c.Query("UPPER_8DAYS") == "true" {
		indicators.UPPER8DAYS = true
	}
	if c.Query("UPPER_9DAYS") == "true" {
		indicators.UPPER9DAYS = true
	}
	if c.Query("UPPER_4DAYS") == "true" {
		indicators.UPPER4DAYS = true
	}
	if c.Query("HEAVEN_RULE") == "true" {
		indicators.HEAVENRULE = true
	}
	if c.Query("UPSIDE_VOLUME") == "true" {
		indicators.UPSIDEVOLUME = true
	}
	if c.Query("BEARISH_ENGULFING") == "true" {
		indicators.BEARISHENGULFING = true
	}
	if c.Query("REVERSING_HAMMER") == "true" {
		indicators.REVERSINGHAMMER = true
	}
	if c.Query("SHOOTING_STAR") == "true" {
		indicators.SHOOTINGSTAR = true
	}
	if c.Query("EVENING_STAR") == "true" {
		indicators.EVENINGSTAR = true
	}
	if c.Query("FIRST_DAWN") == "true" {
		indicators.FIRSTDAWN = true
	}
	if c.Query("PREGNANT") == "true" {
		indicators.PREGNANT = true
	}
	if c.Query("BLACK_CLOUD_TOPS") == "true" {
		indicators.BLACKCLOUDTOPS = true
	}
	if c.Query("MORNING_STAR") == "true" {
		indicators.MORNINGSTAR = true
	}
	if c.Query("NARROW_FINISH") == "true" {
		indicators.NARROWFINISH = true
	}

	// 解析数值类型指标
	if v := c.Query("UPP_DAYS"); v != "" {
		if num, err := strconv.Atoi(v); err == nil {
			indicators.UPP_DAYS = num
		}
	}
	if v := c.Query("CONCERN_RANK_7DAYS"); v != "" {
		if num, err := strconv.Atoi(v); err == nil {
			indicators.CONCERN_RANK_7DAYS = num
		}
	}
	if v := c.Query("UPNDAY"); v != "" {
		if num, err := strconv.Atoi(v); err == nil {
			indicators.UPNDAY = num
		}
	}
	if v := c.Query("DOWNNDAY"); v != "" {
		if num, err := strconv.Atoi(v); err == nil {
			indicators.DOWNNDAY = num
		}
	}

	result := data.NewStockDataApi().GetAllStocks(page, pageSize, keyword, indicators)
	if result == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    gin.H{"result": gin.H{"data": []interface{}{}, "count": 0}},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"result": gin.H{
				"data":        result.Result.Data,
				"count":       result.Result.Count,
				"nextpage":    result.Result.Nextpage,
				"currentpage": result.Result.Currentpage,
			},
		},
	})
}

// GetSponsorInfo 获取赞助信息（综合用户管理和本地配置）
func GetSponsorInfo(c *gin.Context) {
	// 获取本地配置的赞助信息
	level, active := data.EffectiveSponsorVipLevel()

	// 检查用户管理中的 VIP 信息
	userID, _ := middleware.GetUserIDFromContext(c)
	if userID > 0 {
		var user models.User
		if err := db.Dao.Where("id = ?", userID).First(&user).Error; err == nil {
			if user.IsActive {
				role := middleware.GetUserRole(user)
				// 角色是VIP及以上
				if role == middleware.RoleVIP || role == middleware.RoleAdmin || role == middleware.RoleSuperAdmin {
					active = true
					if role == middleware.RoleVIP && level < 2 {
						level = 2
					} else if role == middleware.RoleAdmin && level < 3 {
						level = 3
					} else if role == middleware.RoleSuperAdmin && level < 4 {
						level = 4
					}
				}
				// 检查用户表中的VIP信息
				if !active && user.VipLevel > 0 && user.VipEndAt != nil && !user.VipEndAt.IsZero() {
					if time.Now().Before(*user.VipEndAt) {
						active = true
						if user.VipLevel > level {
							level = user.VipLevel
						}
					}
				}
			}
		}
	}

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
	types := agent.NewCronTaskApi().GetTaskTypes()
	result := make([]map[string]string, len(types))
	for i, t := range types {
		result[i] = map[string]string{
			"name":  t.A,
			"label": t.B,
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// CreateCronTask 创建定时任务（用户隔离：自动关联当前用户）
func CreateCronTask(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var task models.CronTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	task.UserID = userID

	err := agent.NewCronTaskApi().Create(&task)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建失败：" + err.Error()})
		return
	}

	// 同步添加到调度器
	if task.Enable && CronSchedulerHooks.AddCronTask != nil {
		CronSchedulerHooks.AddCronTask(&task)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功"})
}

// SearchCronTasks 搜索定时任务（用户隔离）
func SearchCronTasks(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	keyword := c.DefaultQuery("keyword", "")
	tasks := agent.NewCronTaskApi().SearchTasks(keyword, userID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": tasks})
}

// GetCronTaskList 获取定时任务列表（用户隔离）
func GetCronTaskList(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.DefaultQuery("name", "")
	taskType := c.DefaultQuery("taskType", "")
	status := c.DefaultQuery("status", "")

	query := &models.CronTaskQuery{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		Name:     name,
		TaskType: taskType,
		Status:   status,
	}

	result := agent.NewCronTaskApi().List(query)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// GetCronTaskByID 获取定时任务详情（用户隔离：禁止查看他人任务）
func GetCronTaskByID(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的任务ID"})
		return
	}

	task, err := agent.NewCronTaskApi().GetByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": task})
}

// ExecuteCronTaskNow 立即执行定时任务（用户隔离）
func ExecuteCronTaskNow(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的任务ID"})
		return
	}

	task, err := agent.NewCronTaskApi().GetByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "任务不存在：" + err.Error()})
		return
	}

	go func() {
		err := agent.NewCronTaskApi().ExecuteTask(nil, task)
		if err != nil {
			logger.SugaredLogger.Errorf("执行任务失败：%v %s", err, task.Name)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "任务已开始执行"})
}

// UpdateCronTask 更新定时任务（用户隔离：禁止修改他人任务）
func UpdateCronTask(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var task models.CronTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	task.UserID = userID

	err := agent.NewCronTaskApi().Update(&task)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新失败：" + err.Error()})
		return
	}

	// 同步更新调度器（移除旧 entry，添加新 entry）
	if CronSchedulerHooks.AddCronTask != nil {
		CronSchedulerHooks.AddCronTask(&task)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteCronTask 删除定时任务（用户隔离：禁止删除他人任务）
func DeleteCronTask(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的任务ID"})
		return
	}

	err = agent.NewCronTaskApi().Delete(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除失败：" + err.Error()})
		return
	}

	// 从调度器移除
	if CronSchedulerHooks.RemoveCronTask != nil {
		CronSchedulerHooks.RemoveCronTask(uint(id))
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// EnableCronTask 启用/暂停定时任务（用户隔离：禁止操作他人任务）
func EnableCronTask(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的任务ID"})
		return
	}

	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误"})
		return
	}

	err = agent.NewCronTaskApi().EnableTask(uint(id), req.Enable, userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "操作失败：" + err.Error()})
		return
	}

	// 同步调度器：启用则添加，禁用则移除
	if req.Enable {
		task, _ := agent.NewCronTaskApi().GetByID(uint(id), userID)
		if task != nil && CronSchedulerHooks.AddCronTask != nil {
			CronSchedulerHooks.AddCronTask(task)
		}
	} else {
		if CronSchedulerHooks.RemoveCronTask != nil {
			CronSchedulerHooks.RemoveCronTask(uint(id))
		}
	}

	msg := "已暂停"
	if req.Enable {
		msg = "已启用"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg})
}

// ShareText 分享文本
func ShareText(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": "分享成功"})
}

// ShareAnalysis 分享分析
func ShareAnalysis(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": "分享成功"})
}

// SendDingDingMessage 发送钉钉消息（Web端用户隔离，读取当前用户配置）
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
	// 读取当前用户的钉钉配置，而非全局默认配置
	userID, _ := middleware.GetUserIDFromContext(c)
	cfg := data.GetSettingConfigByUserID(userID)
	if cfg == nil || cfg.Settings == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": "用户配置不存在"})
		return
	}
	result := data.NewDingDingAPI().SendToDingDingWithConfig(title, msg, cfg.DingPushEnable, cfg.DingRobot)
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

// FetchAndSaveMarketStatistic 获取并保存市场统计数据
func FetchAndSaveMarketStatistic(c *gin.Context) {
	err := data.NewMarketStatisticApi().FetchAndSave()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "获取市场统计数据失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
	})
}

// GetTodayMarketStatistic 获取今日市场统计数据
func GetTodayMarketStatistic(c *gin.Context) {
	stats := data.NewMarketStatisticApi().GetTodayData()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// GetRecentDaysMarketStatistic 获取最近N天市场统计数据
func GetRecentDaysMarketStatistic(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 {
		days = 7
	}
	stats := data.NewMarketStatisticApi().GetRecentDaysData(days)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// GetMarketStatisticByDate 按日期获取市场统计数据
func GetMarketStatisticByDate(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "日期参数不能为空",
		})
		return
	}
	stats := data.NewMarketStatisticApi().GetByDate(date)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

// FetchAndSaveBKFundFlow 获取并保存板块资金流向数据
func FetchAndSaveBKFundFlow(c *gin.Context) {
	count, err := data.NewBKFundFlowApi().FetchAndSave()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "获取板块资金流向失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "获取成功",
		"data": gin.H{
			"count": count,
		},
	})
}

// GetBKFundFlowTopListByDate 获取指定日期最新快照的板块资金排名
func GetBKFundFlowTopListByDate(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	topN, _ := strconv.Atoi(c.DefaultQuery("topN", "20"))
	list := data.NewBKFundFlowApi().GetBKFundFlowTopListByDate(date, topN)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
	})
}

// GetBKFundFlowListByDate 获取某个板块指定日期的资金流向历史数据
func GetBKFundFlowListByDate(c *gin.Context) {
	code := c.Query("code")
	date := c.Query("date")
	if code == "" || date == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "code 和 date 参数不能为空",
		})
		return
	}
	points := data.NewBKFundFlowApi().GetBKFundFlowListByDate(code, date)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    points,
	})
}

// GetAllBKCodes 获取所有板块代码和名称
func GetAllBKCodes(c *gin.Context) {
	codes := data.NewBKFundFlowApi().GetAllBKCodes()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    codes,
	})
}

// GetBKFundFlowList 获取板块资金流向历史数据（折线图用）
func GetBKFundFlowList(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "code 参数不能为空",
		})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "240"))
	points := data.NewBKFundFlowApi().GetBKFundFlowList(code, limit)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    points,
	})
}

// GetBKFundFlowTopList 获取最新板块资金排名
func GetBKFundFlowTopList(c *gin.Context) {
	topN, _ := strconv.Atoi(c.DefaultQuery("topN", "20"))
	list := data.NewBKFundFlowApi().GetBKFundFlowTopList(topN)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
	})
}

// GetAllConceptCodes 获取所有概念代码和名称
func GetAllConceptCodes(c *gin.Context) {
	codes := data.NewConceptFundFlowApi().GetAllConceptCodes()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    codes,
	})
}

// GetConceptFundFlowListByDate 获取某个概念指定日期的资金流向历史数据
func GetConceptFundFlowListByDate(c *gin.Context) {
	code := c.Query("code")
	date := c.Query("date")
	if code == "" || date == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "code 和 date 参数不能为空",
		})
		return
	}
	points := data.NewConceptFundFlowApi().GetConceptFundFlowListByDate(code, date)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    points,
	})
}

// GetConceptFundFlowTopListByDate 获取指定日期最新快照的概念资金排名
func GetConceptFundFlowTopListByDate(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	topN, _ := strconv.Atoi(c.DefaultQuery("topN", "20"))
	list := data.NewConceptFundFlowApi().GetConceptFundFlowTopListByDate(date, topN)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
	})
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

// GetHotWords 获取最近24小时热词
// 先确保数据库有最近24小时新闻，再分析（与桌面端用户手动刷新流程一致）
func GetHotWords(c *gin.Context) {
	// 先检查 DB 中是否已有 24h 内新闻
	var newsCount int64
	db.Dao.Model(&models.Telegraph{}).Where("created_at>?", time.Now().Add(-24*time.Hour)).Count(&newsCount)
	if newsCount == 0 {
		// DB 为空时，主动触发爬取（与 RefreshTelegraphList 完全一致）
		logger.SugaredLogger.Info("[GetHotWords] DB无24h新闻，开始爬取...")
		newsApi := data.NewMarketNewsApi()
		newsApi.TelegraphList(30)
		newsApi.GetSinaNews(30)
		db.Dao.Model(&models.Telegraph{}).Where("created_at>?", time.Now().Add(-24*time.Hour)).Count(&newsCount)
		logger.SugaredLogger.Infof("[GetHotWords] 爬取完成，当前24h新闻: %d条", newsCount)
	}
	result, frequencies := data.NewsAnalyze("", false)
	logger.SugaredLogger.Infof("[GetHotWords] DB中24h新闻: %d条, Score: %.2f, 词频数: %d",
		newsCount, result.Score, len(frequencies))
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": map[string]any{
			"result":      result,
			"frequencies": frequencies,
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
	// 使用 GetAllStockChangesWithPaging 获取所有类型的异动数据
	result := data.NewStockChangesApi().GetAllStockChangesWithPaging(500)
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

// ValidateCronExpr 验证 Cron 表达式
func ValidateCronExpr(c *gin.Context) {
	expr := c.DefaultQuery("expr", "")
	if expr == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Cron 表达式无效：表达式不能为空", "valid": false})
		return
	}

	err := agent.NewCronTaskApi().ValidateCronExpr(expr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Cron 表达式无效：" + err.Error(), "valid": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Cron 表达式有效", "valid": true})
}

// CalculateNextRunTimes 计算未来 N 次执行时间
func CalculateNextRunTimes(c *gin.Context) {
	expr := c.DefaultQuery("expr", "")
	if expr == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []string{}})
		return
	}
	count, _ := strconv.Atoi(c.DefaultQuery("count", "5"))
	if count <= 0 {
		count = 5
	}
	times := agent.NewCronTaskApi().CalculateNextRunTimes(expr, count)
	// 将 time.Time 转为字符串
	result := make([]string, 0, len(times))
	for _, t := range times {
		result = append(result, t.Format("2006-01-02 15:04:05"))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// CalculateNextRunTime 计算下次执行时间
func CalculateNextRunTime(c *gin.Context) {
	expr := c.DefaultQuery("expr", "")
	if expr == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
		return
	}
	next := agent.NewCronTaskApi().CalculateNextRunTime(expr)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": next.Format("2006-01-02 15:04:05")})
}

// GetUplimitHot 获取涨停梯队数据（与桌面端 GetUplimitHot 一致）
func GetUplimitHot(c *gin.Context) {
	date := c.DefaultQuery("date", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 {
		limit = 20
	}
	result := data.NewMarketNewsApi().GetUplimitHot(date, limit)
	// 桌面端返回格式: {"code": xxx, "message": "xxx", "data": {...}}
	// 前端期望 code=20000 表示成功
	c.JSON(http.StatusOK, result)
}

// GetChangeRank 获取异动排行（股票/行业/概念异动次数）
func GetChangeRank(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "1"))
	topN, _ := strconv.Atoi(c.DefaultQuery("topN", "20"))
	if days <= 0 {
		days = 1
	}
	if topN <= 0 {
		topN = 20
	}

	result, err := data.NewStockChangeHistoryService().GetChangeRank(days, topN)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"topStocks":     []data.ChangeRankItem{},
				"topIndustries": []data.ChangeRankItem{},
				"topConcepts":   []data.ChangeRankItem{},
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetDailyChangeStats 获取每日异动统计
func GetDailyChangeStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}

	result, err := data.NewStockChangeHistoryService().GetDailyChangeStats(days)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []data.DailyChangeStats{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// GetChangeTypeDailyStats 获取按异动类型的每日统计
func GetChangeTypeDailyStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		days = 30
	}

	result, err := data.NewStockChangeHistoryService().GetChangeTypeDailyStats(days)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []data.ChangeTypeDailyStats{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// GetAllMarkets 获取所有市场
func GetAllMarkets(c *gin.Context) {
	markets, err := data.NewStockDataApi().GetAllMarkets()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": markets})
}

// GetAllIndustries 获取所有行业
func GetAllIndustries(c *gin.Context) {
	industries, err := data.NewStockDataApi().GetAllIndustries()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": industries})
}

// GetAllConcepts 获取所有概念
func GetAllConcepts(c *gin.Context) {
	concepts, err := data.NewStockDataApi().GetAllConcepts()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": concepts})
}

// GetAllSkills 获取所有技能
func GetAllSkills(c *gin.Context) {
	skills := data.NewSkillApi().GetAll()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": skills})
}

// GetSkillList 获取技能列表
func GetSkillList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	name := c.Query("name")
	category := c.Query("category")

	query := &models.SkillQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     name,
		Category: category,
	}
	if enable := c.Query("enable"); enable != "" {
		parsed, err := strconv.ParseBool(enable)
		if err == nil {
			query.Enable = &parsed
		}
	}

	result := data.NewSkillApi().List(query)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// GetSkillByID 根据ID获取技能
func GetSkillByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的技能ID"})
		return
	}

	skill, err := data.NewSkillApi().GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "技能不存在", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": skill})
}

// CreateSkill 创建技能
func CreateSkill(c *gin.Context) {
	var skill models.Skill
	if err := c.ShouldBindJSON(&skill); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if err := data.NewSkillApi().Create(&skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建技能失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": skill})
}

// UpdateSkill 更新技能
func UpdateSkill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的技能ID"})
		return
	}
	var skill models.Skill
	if err := c.ShouldBindJSON(&skill); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	skill.ID = uint(id)
	if err := data.NewSkillApi().Update(&skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "更新技能失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteSkill 删除技能
func DeleteSkill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的技能ID"})
		return
	}
	if err := data.NewSkillApi().Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "删除技能失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// EnableSkill 启用或禁用技能
func EnableSkill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的技能ID"})
		return
	}
	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if err := data.NewSkillApi().EnableSkill(uint(id), req.Enable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "操作失败: " + err.Error()})
		return
	}
	enableMsg := "已禁用"
	if req.Enable {
		enableMsg = "已启用"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": enableMsg})
}

// ImportSkills 批量导入技能（JSON 文件上传或 JSON body）
func ImportSkills(c *gin.Context) {
	var skills []models.Skill

	// 优先尝试读取上传文件
	file, _, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		fileData, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "读取文件失败: " + err.Error()})
			return
		}
		if err := json.Unmarshal(fileData, &skills); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "JSON 格式错误: " + err.Error()})
			return
		}
	} else {
		// 回退到 JSON body
		if err := c.ShouldBindJSON(&skills); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
			return
		}
	}

	if len(skills) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "导入数据为空"})
		return
	}

	created, skipped, err := data.NewSkillApi().ImportSkills(skills)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "导入失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("导入完成：成功 %d 个，跳过（已存在） %d 个", created, skipped),
		"data": gin.H{
			"created": created,
			"skipped": skipped,
		},
	})
}

// ExportAllSkills 导出全部技能为 JSON
func ExportAllSkills(c *gin.Context) {
	skills := data.NewSkillApi().GetAllEnabledAndDisabled()
	if len(skills) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "无技能数据", "data": []models.Skill{}})
		return
	}
	exportData, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "导出失败: " + err.Error()})
		return
	}
	filename := fmt.Sprintf("skills-export-%s.json", time.Now().Format("2006-01-02-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", exportData)
}

// ExportSkillByID 导出单个技能为 JSON
func ExportSkillByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的技能ID"})
		return
	}
	skill, err := data.NewSkillApi().GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "技能不存在"})
		return
	}
	exportData, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "导出失败: " + err.Error()})
		return
	}
	filename := fmt.Sprintf("skill-%s-%s.json", skill.Name, time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/json", exportData)
}

// GetMCPServerList 获取 MCP 服务器列表
func GetMCPServerList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	query := &models.MCPServerQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     c.Query("name"),
		Status:   c.Query("status"),
	}
	if enable := c.Query("enable"); enable != "" {
		parsed, err := strconv.ParseBool(enable)
		if err == nil {
			query.Enable = &parsed
		}
	}
	result := data.NewMCPServerApi().List(query)
	if result == nil {
		result = &models.MCPServerPageResp{Total: 0, Data: []models.MCPServer{}}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// GetMCPServerByID 根据 ID 获取 MCP 服务器
func GetMCPServerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的服务器ID"})
		return
	}
	server, err := data.NewMCPServerApi().GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "服务器不存在", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": server})
}

// CreateMCPServer 创建 MCP 服务器
func CreateMCPServer(c *gin.Context) {
	var server models.MCPServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if err := data.NewMCPServerApi().Create(&server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建服务器失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": server})
}

// UpdateMCPServer 更新 MCP 服务器
func UpdateMCPServer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的服务器ID"})
		return
	}
	var server models.MCPServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	server.ID = uint(id)
	if err := data.NewMCPServerApi().Update(&server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "更新服务器失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteMCPServer 删除 MCP 服务器
func DeleteMCPServer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的服务器ID"})
		return
	}
	if err := data.NewMCPServerApi().Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "删除服务器失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// EnableMCPServer 启用或禁用 MCP 服务器
func EnableMCPServer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的服务器ID"})
		return
	}
	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if err := data.NewMCPServerApi().EnableServer(uint(id), req.Enable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "操作失败: " + err.Error()})
		return
	}
	message := "已禁用"
	if req.Enable {
		message = "已启用"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": message})
}

// TestMCPServer 测试 MCP 服务器连接
func TestMCPServer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的服务器ID"})
		return
	}
	result, err := data.NewMCPServerApi().TestConnection(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "message": err.Error(), "data": result})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": result, "data": result})
}

// GetMCPToolsByServerID 获取指定 MCP 服务器工具
func GetMCPToolsByServerID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的服务器ID"})
		return
	}
	tools := data.NewMCPServerApi().GetToolsByServerID(uint(id))
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": tools})
}

// GetAllMCPTools 获取所有 MCP 工具
func GetAllMCPTools(c *gin.Context) {
	tools := data.NewMCPServerApi().GetAllTools()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": tools})
}

// ==================== 自定义选股策略 ====================

// GetCustomStrategies 获取所有自定义策略（简化接口）
func GetCustomStrategies(c *gin.Context) {
	strategies := data.NewCustomStrategyApi().GetAllCustomStrategies()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": strategies})
}

// GetCustomStrategyList 分页获取自定义策略
func GetCustomStrategyList(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	name := c.DefaultQuery("name", "")

	query := &models.CustomStrategyQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     name,
	}

	result, err := data.NewCustomStrategyApi().GetCustomStrategyList(query)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": "查询失败: " + err.Error()})
		return
	}

	_ = userID // CustomStrategy 目前为公共数据，不做用户隔离
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// SaveCustomStrategy 创建或更新自定义策略
func SaveCustomStrategy(c *gin.Context) {
	var strategy models.CustomStrategy
	if err := c.ShouldBindJSON(&strategy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	result := data.NewCustomStrategyApi().SaveCustomStrategy(strategy)
	if result == "添加成功" || result == "更新成功" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": result})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": result})
	}
}

// DeleteCustomStrategy 删除自定义策略
func DeleteCustomStrategy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的策略ID"})
		return
	}

	result := data.NewCustomStrategyApi().DeleteCustomStrategy(uint(id))
	if result == "删除成功" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": result})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": result})
	}
}
