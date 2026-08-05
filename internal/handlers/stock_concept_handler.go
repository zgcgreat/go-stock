package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
)

// GetConceptList 获取概念标签列表
func GetConceptList(c *gin.Context) {
	concepts := data.NewStockConceptApi(db.Dao).GetConceptList()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    concepts,
	})
}

// AddConcept 新建概念标签
func AddConcept(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Sort int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	ok := data.NewStockConceptApi(db.Dao).AddConcept(data.Concept{Name: req.Name, Sort: req.Sort})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "添加失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "添加成功",
	})
}

// UpdateConcept 修改概念名称
func UpdateConcept(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	ok := data.NewStockConceptApi(db.Dao).UpdateConcept(id, req.Name)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "修改失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "修改成功",
	})
}

// RemoveConcept 删除概念标签
func RemoveConcept(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ok := data.NewStockConceptApi(db.Dao).RemoveConcept(id)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "移除失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "移除成功",
	})
}

// AddStockConcept 把股票加入概念
func AddStockConcept(c *gin.Context) {
	var req struct {
		ConceptId int    `json:"conceptId" binding:"required"`
		StockCode string `json:"stockCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	ok := data.NewStockConceptApi(db.Dao).AddStockConcept(req.ConceptId, req.StockCode)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "添加失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "添加成功",
	})
}

// RemoveStockConcept 把股票移出概念
func RemoveStockConcept(c *gin.Context) {
	var req struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		ConceptId int    `json:"conceptId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	ok := data.NewStockConceptApi(db.Dao).RemoveStockConcept(req.Code, req.Name, req.ConceptId)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "移除失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "移除成功",
	})
}

// GetAllStockConcepts 返回全部概念-股票归属记录（含概念信息）
func GetAllStockConcepts(c *gin.Context) {
	list := data.NewStockConceptApi(db.Dao).GetAllStockConcepts()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
	})
}

// GetStockConceptsByStockCode 查询单只股票所属的概念
func GetStockConceptsByStockCode(c *gin.Context) {
	stockCode := c.Query("stockCode")
	list := data.NewStockConceptApi(db.Dao).GetStockConceptsByStockCode(stockCode)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    list,
	})
}
