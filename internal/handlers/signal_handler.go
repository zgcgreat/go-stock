package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/models"
)

// 004 批2：信号流水持久化端点——解锁桥接层 4 个桩（Save/GetPage/GetStats/Clear）。
// 全部 JWT 保护、按登录用户隔离（userID > 0 过滤）；VIP2 门槛与上游同语义留在客户端
// （canUseSignalMonitor），服务端强制列为可选加固。
// NotifySignal 端点在批 4 提供（/signals/notify）。

// SaveSignalRecordsHandler 信号流水批量落库：POST /signals/records
// body 为 []models.SignalRecord 数组（UserID 字段 json:"-"，请求体传入无效，由服务端注入）。
// 桌面语义：返回空串=成功，否则为错误信息——保持一致，前端不阻塞等待。
func SaveSignalRecordsHandler(c *gin.Context) {
	var items []models.SignalRecord
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "信号数据格式错误: " + err.Error()})
		return
	}
	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
		return
	}
	if err := data.NewSignalRecordService().SaveSignalRecords(items, getUserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}

// GetSignalRecordPageHandler 信号流水分页查询：GET /signals/records
// query: keyword,startTime,endTime(Unix秒),page,pageSize。错误路径与桌面一致返回空页而非 5xx，
// 避免前端面板把瞬时查询失败渲染成报错横幅。
func GetSignalRecordPageHandler(c *gin.Context) {
	query := models.SignalRecordQuery{
		Keyword:   c.Query("keyword"),
		StartTime: parseQueryInt64(c, "startTime"),
		EndTime:   parseQueryInt64(c, "endTime"),
		Page:      int(parseQueryInt64(c, "page")),
		PageSize:  int(parseQueryInt64(c, "pageSize")),
	}
	page, err := data.NewSignalRecordService().GetSignalRecordPage(query, getUserID(c))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": models.SignalRecordPageData{List: []models.SignalRecord{}}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": *page})
}

// ClearSignalRecordsHandler 清空当前用户的信号流水：DELETE /signals/records
// 桌面语义同上：返回空串=成功。
func ClearSignalRecordsHandler(c *gin.Context) {
	if err := data.NewSignalRecordService().ClearSignalRecords(getUserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}

// GetSignalStatsHandler 信号收益统计：GET /signals/stats
// query: startTime,endTime(Unix秒)。错误路径与桌面一致返回空统计。
func GetSignalStatsHandler(c *gin.Context) {
	query := models.SignalStatQuery{
		StartTime: parseQueryInt64(c, "startTime"),
		EndTime:   parseQueryInt64(c, "endTime"),
	}
	res, err := data.NewSignalRecordService().GetSignalStats(query, getUserID(c))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": models.SignalStatResult{Stocks: []models.SignalStatItem{}}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": *res})
}

// parseQueryInt64 query 参数 → int64（缺省/非法返回 0，语义与桌面零值一致）
func parseQueryInt64(c *gin.Context, key string) int64 {
	v, err := strconv.ParseInt(c.Query(key), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// NotifySignalHandler 信号提醒推送出口：POST /signals/notify（004 批4）。
// 渠道语义与桌面 NotifySignal（app_signal_monitor.go）一致：feishu/dingding 服务端转发；
// app 渠道由前端浏览器 Notification API 承担（服务端无法弹系统通知），服务端忽略该渠道。
// 空 channels = 全部渠道（桌面 useAll 语义），服务端按全部外部渠道处理。
// 飞书/钉钉走全局配置（与桌面同源）；per-user webhook 明确不做（004 §5）。
func NotifySignalHandler(c *gin.Context) {
	var req struct {
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Plain    string   `json:"plain"`
		Channels []string `json:"channels"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "推送参数格式错误: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "empty title"})
		return
	}
	useAll := len(req.Channels) == 0
	has := func(channel string) bool {
		if useAll {
			return true
		}
		for _, ch := range req.Channels {
			if ch == channel {
				return true
			}
		}
		return false
	}
	if has("feishu") {
		go data.NewFeishuAPI().SendToFeishu(req.Title, req.Content)
	}
	if has("dingding") {
		go data.NewDingDingAPI().SendToDingDing(req.Title, req.Content)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": "ok"})
}
