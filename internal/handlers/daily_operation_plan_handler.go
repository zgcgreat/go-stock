package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/models"
	"go-stock/internal/middleware"
)

// GetDailyOperationPlanList 分页查询每日操作计划
func GetDailyOperationPlanList(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	query := &models.DailyOperationPlanQuery{}
	query.Page, _ = strconv.Atoi(c.Query("page"))
	query.PageSize, _ = strconv.Atoi(c.Query("pageSize"))
	query.StockCode = c.Query("stockCode")
	query.StockName = c.Query("stockName")
	query.PlanDate = c.Query("planDate")
	query.Status = c.Query("status")

	page, err := data.NewDailyOperationPlanApi().GetDailyOperationPlanList(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "查询每日操作计划失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    page,
	})
}

// GetDailyOperationPlanByID 根据 ID 获取操作计划
func GetDailyOperationPlanByID(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	plan, err := data.NewDailyOperationPlanApi().GetDailyOperationPlanByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "操作计划不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    plan,
	})
}

// SaveDailyOperationPlan 新增或更新操作计划
func SaveDailyOperationPlan(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var plan models.DailyOperationPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	result := data.NewDailyOperationPlanApi().SaveDailyOperationPlan(plan, userID)
	if result != "添加成功" && result != "更新成功" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": result,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": result,
	})
}

// DeleteDailyOperationPlan 根据 ID 删除操作计划
func DeleteDailyOperationPlan(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	result := data.NewDailyOperationPlanApi().DeleteDailyOperationPlan(uint(id), userID)
	if result != "删除成功" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": result,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": result,
	})
}

// UpdateDailyOperationPlanStatus 更新操作计划状态
func UpdateDailyOperationPlanStatus(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := data.NewDailyOperationPlanApi().UpdateDailyOperationPlanStatus(uint(id), req.Status, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// UpdateDailyOperationPlanAlert 更新操作计划盘中预警开关
func UpdateDailyOperationPlanAlert(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req struct {
		EnableAlert bool `json:"enableAlert"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := data.NewDailyOperationPlanApi().UpdateDailyOperationPlanAlert(uint(id), req.EnableAlert, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// GetTdxMinuteTimeData 获取 TDX 分时数据
func GetTdxMinuteTimeData(c *gin.Context) {
	stockCode := c.Query("stockCode")
	if stockCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "缺少 stockCode 参数",
		})
		return
	}

	result := data.NewTdxKLineApi().GetMinuteTimeDataAuto(stockCode)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetAllTdxTransactionData 获取 TDX 成交明细
func GetAllTdxTransactionData(c *gin.Context) {
	stockCode := c.Query("stockCode")
	if stockCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "缺少 stockCode 参数",
		})
		return
	}

	skipCache := c.Query("skipCache") == "true"
	result := data.NewTdxKLineApi().GetAllTransactionDataAuto(stockCode, skipCache)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
