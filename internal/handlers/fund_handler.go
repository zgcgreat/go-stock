package handlers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/internal/middleware"
)

// GetFundList 获取基金列表（fund_basic 公共表，不按用户隔离）
func GetFundList(c *gin.Context) {
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
	var funds []data.FundBasic

	// fund_basic 是公共表（无 user_id 列），不按用户过滤
	query := db.Dao.Model(&data.FundBasic{})

	if queryParam := c.Query("q"); queryParam != "" {
		query = query.Where("code LIKE ? OR name LIKE ? OR full_name LIKE ?",
			"%"+queryParam+"%", "%"+queryParam+"%", "%"+queryParam+"%")
	}

	if code := c.Query("code"); code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}

	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	query.Count(&total)

	err := query.Offset(offset).Limit(pageSize).Find(&funds).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search funds",
			"message": "搜索基金失败",
		})
		return
	}

	totalPages := total / int64(pageSize)
	if total%int64(pageSize) > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       funds,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// FollowFund 关注基金
func FollowFund(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var followReq struct {
		FundCode string `json:"fundCode" binding:"required"`
		FundName string `json:"fundName"`
	}

	if err := c.ShouldBindJSON(&followReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// fundName 未传时自动解析（对齐桌面端 FollowFund 语义：先查本地库，再爬虫兜底）
	fundName := followReq.FundName
	if fundName == "" {
		var fund data.FundBasic
		db.Dao.Where("code=?", followReq.FundCode).First(&fund)
		if fund.Name != "" {
			fundName = fund.Name
		} else if crawled, err := data.NewFundApi().CrawlFundBasic(followReq.FundCode); err == nil && crawled != nil {
			fundName = crawled.Name
		}
	}

	var existingFollow data.FollowedFund
	result := db.Dao.Where("user_id = ? AND fund_code = ?", userID, followReq.FundCode).First(&existingFollow)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Fund already followed",
			"message": "基金已关注",
		})
		return
	}

	newFollow := &data.FollowedFund{
		UserID:   userID,
		Code:     followReq.FundCode,
		Name:     fundName,
		FundCode: followReq.FundCode,
		FundName: fundName,
	}

	if err := db.Dao.Create(newFollow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to follow fund",
			"message": "关注基金失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Fund followed successfully",
		"data":    newFollow,
	})
}

// UnFollowFund 取消关注基金
func UnFollowFund(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	var unfollowReq struct {
		FundCode string `json:"fundCode" binding:"required"`
	}

	if err := c.ShouldBindJSON(&unfollowReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	var existingFollow data.FollowedFund
	result := db.Dao.Where("user_id = ? AND fund_code = ?", userID, unfollowReq.FundCode).First(&existingFollow)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Fund not found in follow list",
			"message": "未关注该基金",
		})
		return
	}

	if err := db.Dao.Delete(&existingFollow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to unfollow fund",
			"message": "取消关注基金失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Fund unfollowed successfully",
	})
}

// GetFundRanking 获取基金排行
func GetFundRanking(c *gin.Context) {
	marketType := c.DefaultQuery("marketType", "kf")
	fundType := c.DefaultQuery("fundType", "all")
	sortField := c.DefaultQuery("sortField", "jnzf")
	sortOrder := c.DefaultQuery("sortOrder", "desc")
	pageIndex, _ := strconv.Atoi(c.DefaultQuery("pageIndex", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))

	result, err := data.NewFundApi().GetFundRanking(marketType, fundType, sortField, sortOrder, pageIndex, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get fund ranking",
			"message": "获取基金排行失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetFollowedFundPaged 分页获取关注的基金列表（对齐桌面端 FollowedFundPagedResult 结构）
func GetFollowedFundPaged(c *gin.Context) {
	userID, _ := middleware.GetUserIDFromContext(c)

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 4
	}
	keyword := c.Query("keyword")

	// keyword 条件同时匹配 legacy 字段（code/name）与 Web 字段（fund_code/fund_name）
	kwCond := "(followed_fund.fund_code LIKE ? OR followed_fund.fund_name LIKE ? OR followed_fund.code LIKE ? OR followed_fund.name LIKE ? OR fund_basic.full_name LIKE ?)"
	kwArgs := []interface{}{"%" + keyword + "%", "%" + keyword + "%", "%" + keyword + "%", "%" + keyword + "%", "%" + keyword + "%"}

	countQuery := db.Dao.Model(&data.FollowedFund{}).Where("user_id = ?", userID)
	if keyword != "" {
		countQuery = countQuery.Joins("LEFT JOIN fund_basic ON fund_basic.code = followed_fund.code").Where(kwCond, kwArgs...)
	}

	var totalCount int64
	if err := countQuery.Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to count followed funds",
			"message": "获取关注基金数量失败",
		})
		return
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	dataQuery := db.Dao.Preload("FundBasic").Where("user_id = ?", userID)
	if keyword != "" {
		dataQuery = dataQuery.Joins("LEFT JOIN fund_basic ON fund_basic.code = followed_fund.code").Where(kwCond, kwArgs...)
	}

	var funds []data.FollowedFund
	offset := (page - 1) * pageSize
	if err := dataQuery.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&funds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch followed funds",
			"message": "获取关注基金列表失败",
		})
		return
	}

	// 兜底：legacy 字段为空的行在前端展示时补齐（组件读取 info.code/info.name）
	for i := range funds {
		if funds[i].Code == "" {
			funds[i].Code = funds[i].FundCode
		}
		if funds[i].Name == "" {
			funds[i].Name = funds[i].FundName
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": data.FollowedFundPagedResult{
			Items:      funds,
			TotalCount: totalCount,
			PageIndex:  page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// GetFundKLineWeb 基金K线（对齐桌面端 App.GetFundKLine → FundKLineApi.GetFundKLineWithFallback）
func GetFundKLineWeb(c *gin.Context) {
	fundCode := c.Param("code")
	klt := c.DefaultQuery("klt", "101")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "600"))

	result := data.NewFundKLineApi().GetFundKLineWithFallback(fundCode, klt, limit)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetFundHistoryNetValueWeb 基金历史净值（对齐桌面端 App.GetFundHistoryNetValue）
func GetFundHistoryNetValueWeb(c *gin.Context) {
	fundCode := c.Param("code")
	pageIndex, _ := strconv.Atoi(c.DefaultQuery("pageIndex", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	res, _ := data.NewFundApi().GetFundHistoryNetValue(fundCode, pageIndex, pageSize, startDate, endDate)
	if res == nil {
		res = []data.FundHistoryNetValue{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    res,
	})
}

// GetFundTop10HoldingsWeb 基金前十大重仓股（对齐桌面端 App.GetFundTop10Holdings）
func GetFundTop10HoldingsWeb(c *gin.Context) {
	res, err := data.NewFundApi().GetFundTop10Holdings(c.Param("code"))
	if err != nil || res == nil {
		res = []data.FundHoldingStock{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    res,
	})
}

// SearchFundCodesWeb 基金代码搜索（对齐桌面端 App.SearchFundCodes）
func SearchFundCodesWeb(c *gin.Context) {
	res := data.NewFundApi().SearchFundCodes(c.Query("keyword"))
	if res == nil {
		res = []data.FundSearchItem{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    res,
	})
}

// GetFollowedFund 获取关注的基金列表
func GetFollowedFund(c *gin.Context) {
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
	var followedFunds []data.FollowedFund

	query := db.Dao.Where("user_id = ?", userID)

	if code := c.Query("code"); code != "" {
		query = query.Where("fund_code LIKE ?", "%"+code+"%")
	}

	if name := c.Query("name"); name != "" {
		query = query.Where("fund_name LIKE ?", "%"+name+"%")
	}

	query.Model(&data.FollowedFund{}).Count(&total)

	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&followedFunds).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch followed funds",
			"message": "获取关注基金列表失败",
		})
		return
	}

	totalPages := total / int64(pageSize)
	if total%int64(pageSize) > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       followedFunds,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}
