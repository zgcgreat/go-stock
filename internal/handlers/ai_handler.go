
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"go-stock/backend/agent"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"
	"go-stock/internal/middleware"
)

// AIAnalyzeRequest AI分析请求
type AIAnalyzeRequest struct {
	StockCode   string `json:"stockCode"`
	StockName   string `json:"stockName"`
	Question    string `json:"question"`
	AIConfigID  int    `json:"aiConfigId"`
	SysPromptID *int   `json:"sysPromptId"`
	Thinking    bool   `json:"thinking"`
}

// AITradeAnalyze AI交易分析 - SSE流式返回
func AITradeAnalyze(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var req AIAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	var questionBuilder strings.Builder
	if req.StockCode != "" {
		questionBuilder.WriteString(fmt.Sprintf("股票代码: %s", req.StockCode))
	}
	if req.StockName != "" {
		questionBuilder.WriteString(fmt.Sprintf(" 股票名称: %s", req.StockName))
	}
	if req.Question != "" {
		questionBuilder.WriteString(fmt.Sprintf("\n问题: %s", req.Question))
	}
	question := questionBuilder.String()
	if question == "" {
		question = "请分析当前市场行情"
	}

	settingConfig := data.GetSettingConfigByUserID(userID)
	if settingConfig == nil || len(settingConfig.AiConfigs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "AI config not found",
			"message": "请先配置AI模型",
		})
		return
	}

	aiConfigID := req.AIConfigID
	if aiConfigID == 0 {
		aiConfigID = int(settingConfig.AiConfigs[0].ID)
	}

	// 获取真实模型名称
	modelName := ""
	aiConfig, found := lo.Find(settingConfig.AiConfigs, func(item *data.AIConfig) bool {
		return int(item.ID) == aiConfigID
	})
	if found {
		modelName = aiConfig.ModelName
	}

	chatID := fmt.Sprintf("web-%d-%d", userID, time.Now().Unix())

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	fmt.Fprintf(c.Writer, "event: chat_id\ndata: %s\n\n", chatID)
	flusher.Flush()

	aiAgent := agent.NewStockAiAgentApi()
	userIDStr := strconv.FormatUint(uint64(userID), 10)
	msgCh := aiAgent.ChatWithContext(c.Request.Context(), question, aiConfigID, req.SysPromptID, false, 10, req.Thinking, "", "", "", userIDStr)

	var fullContent strings.Builder

	for msg := range msgCh {
		if msg == nil {
			continue
		}
		content := msg.Content
		if content != "" {
			fullContent.WriteString(content)
			fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", content)
			flusher.Flush()
		}
	}

	fmt.Fprintf(c.Writer, "event: done\ndata: {\"chatId\":\"%s\"}\n\n", chatID)
	flusher.Flush()

	go func() {
		aiResult := &models.AIResponseResult{
			ChatId:    chatID,
			ModelName: modelName,
			StockCode: req.StockCode,
			StockName: req.StockName,
			Question:  req.Question,
			Content:   fullContent.String(),
			UserID:    userID,
		}
		db.Dao.Create(aiResult)
	}()
}

// GetAIResponses 获取AI分析结果列表
func GetAIResponses(c *gin.Context) {
	// 获取用户ID
	userID, _ := middleware.GetUserIDFromContext(c)

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
	var aiResponses []models.AIResponseResult

	query := db.Dao.Model(&models.AIResponseResult{}).Where("user_id = ?", userID)

	if chatId := c.Query("chatId"); chatId != "" {
		query = query.Where("chat_id LIKE ?", "%"+chatId+"%")
	}
	if stockCode := c.Query("stockCode"); stockCode != "" {
		query = query.Where("stock_code LIKE ?", "%"+stockCode+"%")
	}
	if stockName := c.Query("stockName"); stockName != "" {
		query = query.Where("stock_name LIKE ?", "%"+stockName+"%")
	}
	if question := c.Query("question"); question != "" {
		query = query.Where("question LIKE ?", "%"+question+"%")
	}
	if modelName := c.Query("modelName"); modelName != "" {
		query = query.Where("model_name LIKE ?", "%"+modelName+"%")
	}
	if startDate := c.Query("startDate"); startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate := c.Query("endDate"); endDate != "" {
		// endDate 加一天，用 < 比较，确保包含 endDate 当天全部时间
		endDateParsed, err := time.Parse("2006-01-02", endDate)
		if err == nil {
			nextDay := endDateParsed.AddDate(0, 0, 1).Format("2006-01-02")
			query = query.Where("created_at < ?", nextDay)
		} else {
			query = query.Where("created_at <= ?", endDate+" 23:59:59")
		}
	}

	query.Count(&total)
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&aiResponses).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch AI responses",
			"message": "获取AI分析结果失败",
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
			"list":       aiResponses,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// DeleteAIResponse 删除AI分析结果（用户隔离：禁止删除他人记录）
func DeleteAIResponse(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID format",
			"message": "无效的ID格式",
		})
		return
	}

	// 用户隔离：必须验证记录属于当前用户
	var aiResponse models.AIResponseResult
	result := db.Dao.Where("id = ? AND user_id = ?", uint(id), userID).First(&aiResponse)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "AI response not found",
			"message": "未找到对应的AI分析结果",
		})
		return
	}

	if err := db.Dao.Delete(&aiResponse).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete AI response",
			"message": "删除AI分析结果失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "AI response deleted successfully",
	})
}

// GetAIConfigs 获取AI配置列表（public接口，返回当前用户的AI配置）
// 支持可选认证：登录用户获取自己的配置，未登录用户获取默认配置
func GetAIConfigs(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	settingConfig := data.GetSettingConfigByUserID(userID)
	if settingConfig == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    settingConfig.AiConfigs,
	})
}

// AgentChat Agent模式聊天 - SSE流式（与桌面端 ai-assistant-web/server.go 保持一致）
func AgentChat(c *gin.Context) {
	var req struct {
		Question    string `json:"question" binding:"required"`
		AIConfigID  int    `json:"aiConfigId"`
		SysPromptID *int   `json:"sysPromptId"`
		SessionID   string `json:"sessionId"`
		MemoryCount int    `json:"memoryCount"`
		Thinking    bool   `json:"thinking"`
		AgentMode   string `json:"agentMode"`
		SkillIds    []uint `json:"skillIds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 用户隔离：登录用户获取自己的配置，未登录用户获取默认配置（user_id=0）
	userID, _ := middleware.GetUserIDFromContext(c)
	settingConfig := data.GetSettingConfigByUserID(userID)
	if settingConfig == nil || len(settingConfig.AiConfigs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "AI config not found",
			"message": "请先配置AI模型",
		})
		return
	}

	aiConfigID := req.AIConfigID
	if aiConfigID == 0 {
		aiConfigID = int(settingConfig.AiConfigs[0].ID)
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// 使用 ChatWithContext（与桌面端一致，支持 context 取消和 thinking 模式）
	aiAgent := agent.NewStockAiAgentApi()
	memoryMode := req.SessionID != ""
	memoryCount := req.MemoryCount
	if memoryCount <= 0 {
		memoryCount = 10
	}
	// 传递 userID 作为 optsOverride[2]，使 Agent 内部查询用户自己的配置
	userIDStr := strconv.FormatUint(uint64(userID), 10)

	// 构建用户选择的技能提示词追加（optsOverride[0] = sysPromptOverride）
	var skillPromptOverride string
	if len(req.SkillIds) > 0 {
		var sb strings.Builder
		for _, sid := range req.SkillIds {
			skill, err := data.NewSkillApi().GetByID(sid)
			if err != nil || skill == nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("\n### %s\n", skill.Name))
			if skill.Description != "" {
				sb.WriteString(fmt.Sprintf("描述：%s\n", skill.Description))
			}
			if skill.SystemPrompt != "" {
				sb.WriteString(fmt.Sprintf("%s\n", skill.SystemPrompt))
			}
		}
		skillPromptOverride = sb.String()
	}

	msgCh := aiAgent.ChatWithContext(ctx, req.Question, aiConfigID, req.SysPromptID, memoryMode, memoryCount, req.Thinking, req.AgentMode, skillPromptOverride, "", userIDStr)

	for msg := range msgCh {
		if msg == nil {
			continue
		}

		// 创建符合前端期望的消息对象
		responseData := make(map[string]interface{})
		responseData["role"] = "assistant"

		// 同时发送 reasoning_content 和 content（与桌面端 ai-assistant-web/server.go 保持一致）
		if msg.ReasoningContent != "" {
			responseData["reasoning_content"] = msg.ReasoningContent
		}
		if msg.Content != "" {
			responseData["content"] = msg.Content
		}

		// 检查是否有工具调用
		if len(msg.ToolCalls) > 0 {
			var toolCalls []map[string]interface{}
			for _, tc := range msg.ToolCalls {
				toolCall := map[string]interface{}{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
				toolCalls = append(toolCalls, toolCall)
			}
			responseData["tool_calls"] = toolCalls
		}

		// 发送消息给客户端（与桌面端格式一致：data: {...}\n\n，无 event 行）
		raw, _ := json.Marshal(responseData)
		_, _ = c.Writer.Write([]byte("data: " + string(raw) + "\n\n"))
		flusher.Flush()
	}

	// 发送完成信号（与桌面端格式保持一致：event: done + data: [DONE]）
	_, _ = c.Writer.Write([]byte("event: done\ndata: [DONE]\n\n"))
	flusher.Flush()
}

// GetAIRecommendStocksList 获取AI推荐股票列表（用户隔离）
func GetAIRecommendStocksList(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}

	query := &models.AiRecommendStocksQuery{
		UserID:    userID,
		Page:      page,
		PageSize:  pageSize,
		ModelName: c.Query("modelName"),
		StockName: c.Query("stockName"),
		StockCode: c.Query("stockCode"),
		BkName:    c.Query("bkName"),
		StartDate: c.Query("startDate"),
		EndDate:   c.Query("endDate"),
		EnableAlert: nil,
	}

	// 处理 enableAlert 参数
	if enableAlertStr := c.Query("enableAlert"); enableAlertStr != "" {
		if enableAlertStr == "true" {
			enableAlert := true
			query.EnableAlert = &enableAlert
		} else if enableAlertStr == "false" {
			enableAlert := false
			query.EnableAlert = &enableAlert
		}
	}

	pageData, err := data.NewAiRecommendStocksService().GetAiRecommendStocksList(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch AI recommend stocks",
			"message": "获取AI推荐股票失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       pageData.List,
			"total":      pageData.Total,
			"page":       pageData.Page,
			"pageSize":   pageData.PageSize,
			"totalPages": pageData.TotalPages,
		},
	})
}

// DeleteAIRecommendStock 删除AI推荐股票记录（用户隔离）
func DeleteAIRecommendStock(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID format",
			"message": "无效的ID格式",
		})
		return
	}

	err = data.NewAiRecommendStocksService().DeleteAiRecommendStocks(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete AI recommend stock",
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// UpdateAIRecommendStockAlert 更新AI推荐股票预警状态（用户隔离）
func UpdateAIRecommendStockAlert(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var req struct {
		ID          uint `json:"id" binding:"required"`
		EnableAlert bool `json:"enableAlert"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	err := data.NewAiRecommendStocksService().UpdateAiRecommendStocksAlert(req.ID, userID, req.EnableAlert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update alert status",
			"message": "更新预警状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// GetAiAssistantSessionHandler 获取AI助手会话消息列表（用户隔离）
func GetAiAssistantSessionHandler(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	sessionId := c.Query("sessionId")
	resp, err := data.GetAiAssistantSession(sessionId, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get AI assistant session",
			"message": "获取会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    resp,
	})
}

// SaveAiAssistantSessionHandler 保存AI助手会话消息（用户隔离）
func SaveAiAssistantSessionHandler(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var req struct {
		SessionId string                      `json:"sessionId" binding:"required"`
		Messages  []models.AiAssistantMessage `json:"messages" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	err := data.SaveAiAssistantSession(req.SessionId, userID, req.Messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save AI assistant session",
			"message": "保存会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "保存成功",
	})
}

// 辅助函数：序列化事件数据
func marshalEvent(data map[string]interface{}) string {
	bytes, _ := json.Marshal(data)
	return string(bytes)
}