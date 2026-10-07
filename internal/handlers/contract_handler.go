package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
)

// 005 批5a：合约行情（币安 USDT-M 永续 / Bitget 美股永续）Web 端点。
//
// 桌面端走 app.go 的 Wails 绑定方法，Web 端不引入 Wails 绑定（本项目约定：
// frontend/wailsjs/ 是生成物，跨版本冲突高发），改为在 wails-bridge.js 补 HTTP 回退映射。
// 因此这里只需平铺数据层调用，语义与上游 App 方法一一对应。
//
// 数据源均为公开行情接口，无需用户隔离；但仍挂在 protected 组下统一鉴权，
// 避免未登录用户借此探测服务端出网能力与代理配置状态。

// 合约代理配置（币安/比特大陆/Bitget 各自独立，与全局 HttpProxy 互不影响）
// 单独提供端点而非塞进 /settings：合约页只需这两个字段，
// 避免前端为一页功能拉取并回写整份设置（回写会覆盖其他字段的并发修改）。
type ContractProxy struct {
	BinanceProxy string `json:"binanceProxy"`
	BitgetProxy  string `json:"bitgetProxy"`
}

// GetContractProxyHandler 读取合约专用代理：GET /contracts/proxy
func GetContractProxyHandler(c *gin.Context) {
	cfg := data.GetSettingConfig()
	proxy := ContractProxy{}
	if cfg != nil && cfg.Settings != nil {
		proxy.BinanceProxy = cfg.Settings.BinanceProxy
		proxy.BitgetProxy = cfg.Settings.BitgetProxy
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": proxy})
}

// SetContractProxyHandler 保存合约专用代理：POST /contracts/proxy
// body: {binanceProxy, bitgetProxy}。留空表示该数据源不使用代理。
func SetContractProxyHandler(c *gin.Context) {
	var req ContractProxy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数格式错误: " + err.Error()})
		return
	}
	if err := data.UpdateContractProxies(strings.TrimSpace(req.BinanceProxy), strings.TrimSpace(req.BitgetProxy)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}

// splitCSV 按英文逗号切分并去空，用于 symbols 批量参数
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// queryInt 读取整型 query，缺省或非法时返回 def
func queryInt(c *gin.Context, key string, def int) int {
	if v, err := strconv.Atoi(c.Query(key)); err == nil {
		return v
	}
	return def
}

// GetBinanceFuturesSymbolsHandler 币安永续品种列表：GET /contracts/binance/symbols
func GetBinanceFuturesSymbolsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.BinanceSymbolList()})
}

// GetBinanceFuturesTickerHandler 币安永续 24h 行情：GET /contracts/binance/ticker
// query: symbols（英文逗号分隔，为空返回全部）
func GetBinanceFuturesTickerHandler(c *gin.Context) {
	api := data.NewBinanceFuturesApi()
	list := strings.TrimSpace(c.Query("symbols"))
	if list == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": api.GetAllTickers()})
		return
	}
	out := make([]data.BinanceTicker24h, 0)
	for _, s := range splitCSV(list) {
		if t := api.GetTicker24h(s); t != nil {
			out = append(out, *t)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": out})
}

// GetBinanceFuturesPremiumHandler 币安标记价与当期资金费率（榜单排序用）：
// GET /contracts/binance/premium
func GetBinanceFuturesPremiumHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.NewBinanceFuturesApi().GetAllPremiumIndex()})
}

// GetBinanceFuturesDerivativesHandler 币安永续衍生指标（资金费率/未平仓量/多空比 + 历史）：
// GET /contracts/binance/derivatives?symbol=&period=&limit=
func GetBinanceFuturesDerivativesHandler(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 symbol 参数"})
		return
	}
	period := c.DefaultQuery("period", "1h")
	limit := queryInt(c, "limit", 120)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success",
		"data": data.NewBinanceFuturesApi().GetDerivatives(symbol, period, limit)})
}

// GetBinanceFundingRateHistoryHandler 币安资金费率历史：
// GET /contracts/binance/funding-rate?symbol=&limit=
func GetBinanceFundingRateHistoryHandler(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 symbol 参数"})
		return
	}
	limit := queryInt(c, "limit", 100)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success",
		"data": data.NewBinanceFuturesApi().GetFundingRateHistory(symbol, limit)})
}

// GetBitgetFuturesSymbolsHandler Bitget 美股永续品种列表：GET /contracts/bitget/symbols
func GetBitgetFuturesSymbolsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.BitgetSymbolList()})
}

// GetBitgetFuturesTickerHandler Bitget 美股永续 24h 行情：GET /contracts/bitget/ticker
// query: symbols（英文逗号分隔，为空返回全部）
func GetBitgetFuturesTickerHandler(c *gin.Context) {
	api := data.NewBitgetFuturesApi()
	list := strings.TrimSpace(c.Query("symbols"))
	if list == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": api.GetAllTickers()})
		return
	}
	out := make([]data.BitgetTicker, 0)
	for _, s := range splitCSV(list) {
		if t := api.GetTicker(s); t != nil {
			out = append(out, *t)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": out})
}

// GetBitgetFuturesDerivativesHandler Bitget 永续衍生指标：GET /contracts/bitget/derivatives?symbol=
func GetBitgetFuturesDerivativesHandler(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 symbol 参数"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success",
		"data": data.NewBitgetFuturesApi().GetDerivatives(symbol)})
}

// GetBitgetFundingRateHistoryHandler Bitget 资金费率历史：
// GET /contracts/bitget/funding-rate?symbol=&limit=
func GetBitgetFundingRateHistoryHandler(c *gin.Context) {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 symbol 参数"})
		return
	}
	limit := queryInt(c, "limit", 100)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success",
		"data": data.NewBitgetFuturesApi().GetFundingRateHistory(symbol, limit)})
}
