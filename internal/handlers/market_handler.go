package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
)

// GetMarketTradingTimeHandler 交易时段查询：GET /market/trading-time?market=a|hk|us
//
// 004 批1：为信号监控引擎的 anyMarketOpen 提供桌面 IsTradingTime 系列的 Web 等价物——
// 原桥接层在 Web 模式硬编码返回 false，会静默杀死引擎的全部扫描（anyMarketOpen 恒 false）。
// 实现复用 backend/data/market_hours.go（与桌面 App 方法同一份口径，含节假日校验）。
// 时区口径与桌面一致：A股/港股按东八区取 now，美股由函数内部自转纽约时区。
func GetMarketTradingTimeHandler(c *gin.Context) {
	market := c.Query("market")
	var trading bool
	switch market {
	case "a":
		trading = data.IsTradingTime(nowInShanghai())
	case "hk":
		trading = data.IsHKTradingTime(nowInShanghai())
	case "us":
		trading = data.IsUSTradingTime(time.Now())
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "market 参数仅支持 a/hk/us",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"market":  market,
			"trading": trading,
		},
	})
}

// nowInShanghai 东八区当前时间（与桌面 App 方法的 LoadLocation+FixedZone 兜底口径一致）
func nowInShanghai() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
	return time.Now().In(loc)
}
