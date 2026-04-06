package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// WailsCompatMiddleware Wails兼容中间件，将Web API转换为Wails风格的调用
func WailsCompatMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否为Wails风格的请求
		if strings.HasPrefix(c.Request.URL.Path, "/wails/") {
			handleWailsRequest(c)
			return
		}

		c.Next()
	}
}

func handleWailsRequest(c *gin.Context) {
	// 解析Wails请求格式: /wails/{struct}/{method}
	// 例如: /wails/App/GetStockMinutePriceData
	pathParts := strings.Split(strings.TrimPrefix(c.Request.URL.Path, "/wails/"), "/")
	if len(pathParts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid Wails request format",
		})
		return
	}

	structName := pathParts[0]
	methodName := pathParts[1]

	// 将Wails请求映射到对应的Web API
	switch structName {
	case "App":
		handleAppMethods(c, methodName)
	default:
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "Wails struct not implemented",
		})
	}
}

func handleAppMethods(c *gin.Context, method string) {
	switch method {
	case "GetStockMinutePriceData":
		// 映射到标准Web API
		stockCode := c.Query("stockCode")
		if stockCode == "" {
			// 尝试从POST body中获取
			var reqBody map[string]interface{}
			if err := c.ShouldBindJSON(&reqBody); err == nil {
				if code, ok := reqBody["stockCode"].(string); ok {
					stockCode = code
				}
			}
		}

		if stockCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "stockCode is required",
			})
			return
		}

		// 重写请求路径并转发到对应的Web API
		c.Request.URL.Path = "/api/v1/stocks/" + stockCode + "/minute"
		c.Params = append(c.Params, gin.Param{Key: "code", Value: stockCode})

		// 不直接重定向，而是调用对应的处理函数
		c.Abort() // 终止当前中间件链
		return

	case "GetStockRealTimeData":
		// 映射到实时数据API
		codes := c.Query("codes")
		if codes == "" {
			var reqBody map[string]interface{}
			if err := c.ShouldBindJSON(&reqBody); err == nil {
				if codeParam, ok := reqBody["codes"].(string); ok {
					codes = codeParam
				}
			}
		}

		if codes == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "codes are required",
			})
			return
		}

		// 重写请求路径并转发
		c.Request.URL.Path = "/api/v1/stocks/realtime"
		c.Request.URL.RawQuery = "codes=" + codes

		c.Abort() // 终止当前中间件链
		return

	default:
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "Method not implemented",
		})
		return
	}
}