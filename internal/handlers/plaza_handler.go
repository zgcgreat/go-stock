package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/agent"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"
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

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    prompt,
	})
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
func SyncPlazaData(c *gin.Context) {
	apiBase := c.Query("apiBase")
	username := c.DefaultQuery("username", "Joy")
	password := c.DefaultQuery("password", "zgc@202123")

	sync := data.NewPlazaSyncService(apiBase)

	// 登录
	if err := sync.Login(username, password); err != nil {
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
	}
	c.ShouldBindJSON(&req)

	if req.CronExpr == "" {
		req.CronExpr = "0 0 2 * * *" // 默认每天凌晨2点执行
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

	task := &models.CronTask{
		Name:        "提示词广场数据同步",
		CronExpr:    req.CronExpr,
		TaskType:    "prompt_plaza_sync",
		Target:      "plaza",
		Params:      `{"username":"Joy","password":"zgc@202123"}`,
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

// parseIntWithDefault 安全解析整数
func parseIntWithDefault(s string, defaultVal int) int {
	val, err := strconv.Atoi(s)
	if err != nil || val <= 0 {
		return defaultVal
	}
	return val
}