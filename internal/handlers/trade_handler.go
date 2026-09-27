package handlers

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/services"
	"go-stock/internal/middleware"
)

// getUserID 统一从 context 获取用户 ID，未登录返回 0
func getUserID(c *gin.Context) uint {
	userID, _ := middleware.GetUserIDFromContext(c)
	return userID
}

// GetTradeRecords 获取交易记录列表（用户隔离）
func GetTradeRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")
	direction := c.Query("direction")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	userID := getUserID(c)

	svc := services.GetTradingService()
	result, err := svc.GetTradingRecordList(data.TradingRecordListQuery{
		UserID:    userID,
		Page:      page,
		PageSize:  pageSize,
		Keyword:   keyword,
		Direction: direction,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "获取交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// CreateTradeRecord 创建交易记录（用户隔离）
func CreateTradeRecord(c *gin.Context) {
	var record data.TradingRecord

	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if record.TradingTime.IsZero() {
		record.TradingTime = time.Now()
	}

	record.UserID = getUserID(c)

	svc := services.GetTradingService()
	id, err := svc.AddTradingRecord(record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "创建交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    gin.H{"id": id},
	})
}

// UpdateTradeRecord 更新交易记录（用户隔离：禁止修改他人记录）
func UpdateTradeRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "无效的ID格式",
		})
		return
	}

	var record data.TradingRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	record.ID = uint(id)
	record.UserID = getUserID(c)

	svc := services.GetTradingService()
	if err := svc.UpdateTradingRecord(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "更新交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DeleteTradeRecord 删除交易记录（用户隔离：禁止删除他人记录）
func DeleteTradeRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "无效的ID格式",
		})
		return
	}

	userID := getUserID(c)

	svc := services.GetTradingService()
	if err := svc.DeleteTradingRecord(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "删除交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// ImportTradingRecordsHandler 导入交易记录（Web 端 multipart 上传）。
// 前端上传券商导出的成交记录文件（.xls/.xlsx/.txt/.csv），落临时文件后
// 复用桌面端 ImportTradingRecords 全套解析/去重逻辑（解析器按内容嗅探，不依赖扩展名）。
func ImportTradingRecordsHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "请上传交易记录文件",
		})
		return
	}

	tmp, err := os.CreateTemp("", "trading-import-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "创建临时文件失败: " + err.Error(),
		})
		return
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	if err := c.SaveUploadedFile(fileHeader, tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "保存上传文件失败: " + err.Error(),
		})
		return
	}

	result, err := data.NewStockDataApi().ImportTradingRecords(tmpPath, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "导入交易记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// ExportTradingRecordTemplateHandler 下载交易记录导入模板（.xlsx，含使用说明与示例数据）。
func ExportTradingRecordTemplateHandler(c *gin.Context) {
	xlsxData, err := data.NewStockDataApi().TradingRecordTemplateXLSX()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "生成模板失败: " + err.Error(),
		})
		return
	}
	c.Header("Content-Disposition", `attachment; filename="trading_record_template.xlsx"; filename*=UTF-8''`+url.PathEscape("交易记录导入模板.xlsx"))
	c.Data(http.StatusOK, xlsxContentType, xlsxData)
}

// ExportTableXLSXHandler 通用表格 Excel 导出（Web 端）：前端传列定义 + 行数据，
// 后端用 excelize 生成 .xlsx 附件流返回，由浏览器触发下载（桌面端走保存对话框）。
func ExportTableXLSXHandler(c *gin.Context) {
	var req struct {
		FileName string              `json:"fileName"`
		Table    data.ExportTableData `json:"table"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "参数错误: " + err.Error(),
		})
		return
	}
	xlsxData, err := data.NewStockDataApi().BuildTableXLSX(req.Table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "生成 Excel 失败: " + err.Error(),
		})
		return
	}
	fileName := req.FileName
	if fileName == "" {
		fileName = "导出数据.xlsx"
	}
	c.Header("Content-Disposition", `attachment; filename="export.xlsx"; filename*=UTF-8''`+url.PathEscape(fileName))
	c.Data(http.StatusOK, xlsxContentType, xlsxData)
}
