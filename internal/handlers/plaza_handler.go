package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"

	"go-stock/backend/agent"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"
	"go-stock/backend/security"
	"go-stock/internal/middleware"
)

// GetPlazaPrompts 获取提示词广场本地缓存列表
func GetPlazaPrompts(c *gin.Context) {
	query := &models.PlazaPromptQuery{
		Page:     parseIntWithDefault(c.Query("page"), 1),
		PageSize: parseIntWithDefault(c.Query("pageSize"), 12),
		Category: c.Query("category"),
		Keyword:  c.Query("keyword"),
		Sort:     c.Query("sort"),
		VipOnly:  c.Query("vipOnly"),
	}

	if query.Sort == "" {
		query.Sort = "latest"
	}

	list, total, err := data.GetPlazaPrompts(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "获取提示词列表失败: " + err.Error(),
		})
		return
	}

	pageSize := query.PageSize
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       list,
			"total":      total,
			"page":       query.Page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// GetPlazaPromptDetail 获取提示词广场详情
func GetPlazaPromptDetail(c *gin.Context) {
	extIDStr := c.Param("extId")
	extID, err := strconv.ParseUint(extIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "无效的ID",
		})
		return
	}

	prompt, err := data.GetPlazaPromptByExtID(uint(extID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "提示词不存在",
		})
		return
	}

	// 服务端 VIP 校验：VIP 专属内容对非 VIP 用户不返回正文（前端遮罩仅作 UX，不能作为安全边界）
	if (prompt.VipOnly || prompt.NeedVip) && !isCurrentUserVip(c) {
		prompt.Content = "" // 拦截：不向非 VIP 泄露正文
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    prompt,
	})
}

// isCurrentUserVip 根据当前登录用户本地 VIP 等级与有效期判断是否为有效 VIP。
// Web 端以本地用户 VIP 信息为准（与前端 isLocalVip 逻辑一致）。
func isCurrentUserVip(c *gin.Context) bool {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		return false
	}
	var user models.User
	if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
		return false
	}
	// 管理员视同 VIP（与前端 isLocalVip 保持一致）
	if user.Role == "admin" || user.Role == "super_admin" {
		return true
	}
	if user.VipLevel <= 0 {
		return false
	}
	if user.VipEndAt != nil && !user.VipEndAt.IsZero() && user.VipEndAt.Before(time.Now()) {
		return false
	}
	return true
}

// GetPlazaCategories 获取提示词分类列表
func GetPlazaCategories(c *gin.Context) {
	categories := data.GetPlazaCategories()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    categories,
	})
}

// GetPlazaQuestions 获取问答广场本地缓存列表
func GetPlazaQuestions(c *gin.Context) {
	query := &models.PlazaQuestionQuery{
		Page:     parseIntWithDefault(c.Query("page"), 1),
		PageSize: parseIntWithDefault(c.Query("pageSize"), 15),
		Keyword:  c.Query("keyword"),
		Resolved: c.Query("resolved"),
	}

	list, total, err := data.GetPlazaQuestions(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "获取问答列表失败: " + err.Error(),
		})
		return
	}

	pageSize := query.PageSize
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       list,
			"total":      total,
			"page":       query.Page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// GetPlazaQuestionDetail 获取问答广场详情
func GetPlazaQuestionDetail(c *gin.Context) {
	extIDStr := c.Param("extId")
	extID, err := strconv.ParseUint(extIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "无效的ID",
		})
		return
	}

	question, err := data.GetPlazaQuestionByExtID(uint(extID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "问题不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    question,
	})
}

// SyncPlazaData 手动触发提示词广场数据同步
// 凭据通过 JSON body 传递，避免落入 URL/访问日志。
func SyncPlazaData(c *gin.Context) {
	var req struct {
		ApiBase  string `json:"apiBase"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	c.ShouldBindJSON(&req)
	// 向后兼容：body 解析失败时回退 query（不推荐，凭据会进入访问日志）
	if req.ApiBase == "" {
		req.ApiBase = c.Query("apiBase")
	}
	if req.Username == "" {
		req.Username = c.Query("username")
	}
	if req.Password == "" {
		req.Password = c.Query("password")
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "username 和 password 参数不能为空",
		})
		return
	}

	// SSRF 防护：apiBase 由客户端传入，需校验目标不是内网/保留地址
	if req.ApiBase != "" {
		parsedBase, err := url.Parse(req.ApiBase)
		if err != nil || (parsedBase.Scheme != "http" && parsedBase.Scheme != "https") {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    1,
				"message": "apiBase 必须是 http 或 https 协议",
			})
			return
		}
		if isSSRFBlockedHost(parsedBase.Hostname()) {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    1,
				"message": "不允许访问内网地址",
			})
			return
		}
	}

	sync := data.NewPlazaSyncService(req.ApiBase)

	// 登录
	if err := sync.Login(req.Username, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "登录提示词广场失败: " + err.Error(),
		})
		return
	}

	// 同步提示词
	promptCount, err := sync.SyncPrompts()
	promptMsg := ""
	if err != nil {
		promptMsg = "提示词同步失败: " + err.Error()
	} else {
		promptMsg = "提示词同步成功，共 " + strconv.Itoa(promptCount) + " 条"
	}

	// 同步问答
	questionCount, err := sync.SyncQuestions()
	questionMsg := ""
	if err != nil {
		questionMsg = "问答同步失败: " + err.Error()
	} else {
		questionMsg = "问答同步成功，共 " + strconv.Itoa(questionCount) + " 条"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"promptCount":   promptCount,
			"questionCount": questionCount,
			"promptMsg":     promptMsg,
			"questionMsg":   questionMsg,
		},
	})
}

// CreatePlazaSyncCronTask 快捷创建提示词广场同步定时任务
func CreatePlazaSyncCronTask(c *gin.Context) {
	var req struct {
		CronExpr string `json:"cronExpr"` // 定时表达式，默认每天凌晨2点
		Username string `json:"username"`
		Password string `json:"password"`
	}
	c.ShouldBindJSON(&req)

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "username 和 password 不能为空",
		})
		return
	}

	if req.CronExpr == "" {
		req.CronExpr = "0 0 2 * * *" // 默认每天凌晨2点执行
	}

	// 凭据加密后落库，避免明文密码写入 cron_tasks.params
	encPassword, err := security.EncryptString(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "凭据加密失败: " + err.Error(),
		})
		return
	}

	// 检查是否已存在
	cronApi := agent.NewCronTaskApi()
	if cronApi.ExistsByTaskType("prompt_plaza_sync") {
		c.JSON(http.StatusConflict, gin.H{
			"code":    1,
			"message": "提示词广场同步任务已存在",
		})
		return
	}

	// 归属当前用户，避免创建 user_id=0 的全局任务被他人看到/抢占
	userID, _ := middleware.GetUserIDFromContext(c)

	task := &models.CronTask{
		Name:        "提示词广场数据同步",
		CronExpr:    req.CronExpr,
		TaskType:    "prompt_plaza_sync",
		Target:      "plaza",
		UserID:      userID,
		Params:      fmt.Sprintf(`{"username":"%s","password":"%s"}`, req.Username, encPassword),
		Enable:      true,
		Status:      "active",
		Description: "定时同步提示词广场和问答广场数据到本地数据库",
	}

	if err := cronApi.Create(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "创建同步任务失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    task,
	})
}

// GetPlazaSyncStatus 获取提示词广场同步状态
func GetPlazaSyncStatus(c *gin.Context) {
	lastSyncTime := data.GetLastPlazaSyncTime()

	var promptCount int64
	db.Dao.Model(&models.PlazaPrompt{}).Count(&promptCount)

	var questionCount int64
	db.Dao.Model(&models.PlazaQuestion{}).Count(&questionCount)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"lastSyncTime":  lastSyncTime,
			"promptCount":   promptCount,
			"questionCount": questionCount,
		},
	})
}

// PlazaProxy 服务端代理提示词/技能广场的外部 API 请求。
// Web 模式下浏览器直连外部 HTTP 会被混合内容拦截且存在 CORS 问题，
// 这里由服务端转发，请求始终同源。目标地址需通过内网/保留地址校验（SSRF 防护）。
func PlazaProxy(c *gin.Context) {
	var req struct {
		Method  string            `json:"method"`
		ApiBase string            `json:"apiBase"`
		Path    string            `json:"path"`
		Query   map[string]string `json:"query"`
		Body    string            `json:"body"`
		Token   string            `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	method := strings.ToUpper(req.Method)
	if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "不支持的请求方法"})
		return
	}

	base := strings.TrimSpace(req.ApiBase)
	if base == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "apiBase 不能为空"})
		return
	}
	parsedBase, err := url.Parse(base)
	if err != nil || (parsedBase.Scheme != "http" && parsedBase.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "apiBase 必须是 http 或 https 协议"})
		return
	}
	// SSRF 防护：禁止代理到内网/保留地址
	if isSSRFBlockedHost(parsedBase.Hostname()) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "不允许访问内网地址"})
		return
	}

	target := base
	if !strings.HasPrefix(req.Path, "/") {
		target += "/"
	}
	target += req.Path

	client := resty.New()
	client.SetRedirectPolicy(resty.NoRedirectPolicy()) // 禁止跟随 302，防止绕过 SSRF 校验
	request := client.R().SetHeader("Content-Type", "application/json")
	if req.Token != "" {
		request.SetHeader("Authorization", "Bearer "+req.Token)
	}
	for k, v := range req.Query {
		request.SetQueryParam(k, v)
	}
	if req.Body != "" {
		request.SetBody(req.Body)
	}

	resp, err := request.Execute(method, target)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 1, "message": "广场服务请求失败: " + err.Error()})
		return
	}
	c.Data(resp.StatusCode(), "application/json; charset=utf-8", resp.Body())
}

// parseIntWithDefault 安全解析整数
func parseIntWithDefault(s string, defaultVal int) int {
	val, err := strconv.Atoi(s)
	if err != nil || val <= 0 {
		return defaultVal
	}
	return val
}