package handlers

// okx_handler.go — 007 同步：OKX（欧易）交易 Web 端接线。
// 桌面端走 app_okx.go 的 Wails 绑定，Web 端在 REST 上提供同语义端点；
// 数据层（backend/data/okx_*.go）与桌面共享同一份实现，包括设置页交易开关、
// 模拟盘/实盘双环境与 IP 白名单等约束，服务端不重复鉴权闸门。
//
// 错误约定与桌面一致：查询失败返回空结构（前端面板渲染空态而非报错横幅），
// 下单/撤单失败把错误信息带进结果结构（前端弹提示）。

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/models"
)

// OKXTradingOverview 合约行情总览：SWAP 合约规格 + 24h 行情合并下发。
type OKXTradingOverview struct {
	Instruments []data.OKXInstrument `json:"instruments"`
	Tickers     []data.OKXTicker     `json:"tickers"`
}

// okxTradingOverviewFromInsts 品种过滤 + 行情合并的公共逻辑；品种表为空时回退公共合约表。
func okxTradingOverviewFromInsts(insts []data.OKXInstrument) OKXTradingOverview {
	if insts == nil {
		insts = data.GetOKXInstruments("SWAP")
	}
	byId := make(map[string]struct{}, len(insts))
	outInsts := make([]data.OKXInstrument, 0, len(insts))
	for _, ins := range insts {
		if ins.InstType != "SWAP" {
			continue
		}
		if _, dup := byId[ins.InstId]; dup {
			continue
		}
		byId[ins.InstId] = struct{}{}
		outInsts = append(outInsts, ins)
	}
	tickers := make([]data.OKXTicker, 0, len(outInsts))
	for _, ins := range outInsts {
		if t := data.GetOKXTicker(ins.InstId); t != nil {
			tickers = append(tickers, *t)
		}
	}
	return OKXTradingOverview{Instruments: outInsts, Tickers: tickers}
}

// GetOKXTradingOverviewHandler GET /okx/trading-overview?simulated=
// 拉取当前账户可交易品种（优先私有接口，失败回退公共合约表）与 24h 行情。
func GetOKXTradingOverviewHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	if simulated {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": okxTradingOverviewFromInsts(data.GetOKXAccountInstruments(""))})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": okxTradingOverviewFromInsts(data.GetOKXAccountInstrumentsForEnv(false, ""))})
}

// SetOKXLeverageHandler PUT /okx/leverage {simulated, instId, lever}
// 按环境设置合约杠杆（cross 全仓）。仅 SWAP 永续有效；lever 取值 "1"~"125"。
func SetOKXLeverageHandler(c *gin.Context) {
	var req struct {
		Simulated bool   `json:"simulated"`
		InstId    string `json:"instId" binding:"required"`
		Lever     string `json:"lever" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}
	res, err := data.SetOKXLeverageEnv(req.Simulated, req.InstId, req.Lever)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// GetOKXCandleListHandler GET /okx/candles?instId=&bar=&limit=
// OKX 公共 K 线（含币安/Bitget 没有的品种）。
func GetOKXCandleListHandler(c *gin.Context) {
	instId := c.Query("instId")
	bar := c.Query("bar")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 500
	}
	// afterMs 分页游标：空=取最新一段；非 0=取该毫秒时间戳之后的更早历史（图表向左拖动）
	afterMs, _ := strconv.ParseInt(c.Query("afterMs"), 10, 64)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.GetOKXCandles(instId, bar, limit, afterMs)})
}

// GetOKXTickersHandler GET /okx/tickers?instType=
func GetOKXTickersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.GetOKXTickers(c.Query("instType"))})
}

// GetOkxInstrumentsHandler GET /okx/instruments
// 公共品种表（SPOT+SWAP，免签公共接口 + 6h 缓存），前端按需过滤。
// 供 K 线分析页搜索联想使用：不请求私有账户接口、不拉 24h 行情，轻量快速；
// OKX 不可达时返回空表（前端独立降级，不影响币安/Bitget 联想）。
func GetOkxInstrumentsHandler(c *gin.Context) {
	insts := data.GetOKXInstruments("")
	if insts == nil {
		insts = []data.OKXInstrument{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": insts})
}

// GetOKXTickerHandler GET /okx/ticker?instId=
func GetOKXTickerHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.GetOKXTicker(c.Query("instId"))})
}

// GetOKXRankingsHandler GET /okx/rankings?kind=&limit=
// 行情榜单（涨幅榜/资金费率/多空比等，由 kind 区分）。
func GetOKXRankingsHandler(c *gin.Context) {
	kind := c.Query("kind")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.GetOKXRankings(kind, limit)})
}

// GetOKXWalletHandler GET /okx/wallet?simulated= 账户余额（按环境）。
func GetOKXWalletHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	bal, err := data.GetOKXBalanceForEnv(simulated)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": bal})
}

// GetOKXPositionListHandler GET /okx/positions?simulated=&instId= 持仓列表（按环境）。
func GetOKXPositionListHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	list, err := data.GetOKXPositionListForEnv(simulated, c.Query("instId"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error(), "data": []data.OKXPosition{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// GetOKXPositionTPSLHandler GET /okx/positions/tpsl?simulated=
// 当前持仓的止盈止损线视图（与持仓监控同款口径；未开启止盈止损时返回空数组）。
func GetOKXPositionTPSLHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	views := data.GetOKXPositionTPSLForEnv(simulated)
	if views == nil {
		views = []data.OKXPositionTPSLView{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": views})
}

// GetOKXOpenOrdersHandler GET /okx/orders/open?simulated= 当前挂单（按环境）。
func GetOKXOpenOrdersHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	list, err := data.GetOKXOpenOrdersForEnv(simulated)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error(), "data": []data.OKXOrder{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// GetOKXHistoryOrdersHandler GET /okx/orders/history?simulated= 历史订单（按环境）。
func GetOKXHistoryOrdersHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	list, err := data.GetOKXHistoryOrdersForEnv(simulated)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error(), "data": []data.OKXOrder{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// GetOKXFillsHistoryHandler GET /okx/fills?simulated=&instType=&instId=&limit= 成交明细（按环境）。
func GetOKXFillsHistoryHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 50
	}
	list, err := data.GetOKXFillsHistoryForEnv(simulated, c.Query("instType"), c.Query("instId"), limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error(), "data": []data.OKXTradeFill{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// SubmitOKXOrderHandler POST /okx/orders {simulated, ...OKXOrderRequest}
// 手动下单：受设置页交易开关与模拟盘开关控制（数据层闸门）。
func SubmitOKXOrderHandler(c *gin.Context) {
	var req struct {
		Simulated    bool                `json:"simulated"`
		OrderRequest data.OKXOrderRequest `json:"orderRequest"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 兼容直接平铺 OKXOrderRequest 字段的请求体
		var flat struct {
			Simulated bool `json:"simulated"`
			data.OKXOrderRequest
		}
		if flatErr := c.ShouldBindJSON(&flat); flatErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
			return
		}
		res, err := data.SubmitOKXOrderForEnv(flat.Simulated, &flat.OKXOrderRequest)
		respondOKXOrderResult(c, res, err)
		return
	}
	res, err := data.SubmitOKXOrderForEnv(req.Simulated, &req.OrderRequest)
	respondOKXOrderResult(c, res, err)
}

// respondOKXOrderResult 下单/撤单结果的统一响应：错误信息进 result 由前端弹提示。
func respondOKXOrderResult(c *gin.Context, res *data.OKXPlaceOrderResult, err error) {
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": res})
}

// RevokeOKXOrderHandler DELETE /okx/orders {simulated, instId, ordId} 撤单（按环境）。
func RevokeOKXOrderHandler(c *gin.Context) {
	var req struct {
		Simulated bool   `json:"simulated"`
		InstId    string `json:"instId" binding:"required"`
		OrdId     string `json:"ordId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}
	res, err := data.RevokeOKXOrderForEnv(req.Simulated, req.InstId, req.OrdId)
	respondOKXOrderResult(c, res, err)
}

// CloseOKXPositionHandler POST /okx/positions/close {simulated, instId, posSide, pos} 市价平仓（按环境）。
func CloseOKXPositionHandler(c *gin.Context) {
	var req struct {
		Simulated bool   `json:"simulated"`
		InstId    string `json:"instId" binding:"required"`
		PosSide   string `json:"posSide"`
		Pos       string `json:"pos"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误: " + err.Error()})
		return
	}
	res, err := data.CloseOKXPositionForEnv(req.Simulated, req.InstId, req.PosSide, req.Pos)
	respondOKXOrderResult(c, res, err)
}

// GetOKXAccountConfigHandler GET /okx/account-config?simulated= 账户配置·持仓模式（按环境）。
func GetOKXAccountConfigHandler(c *gin.Context) {
	simulated := defaultTrue(c.Query("simulated"))
	cfg, err := data.GetOKXAccountConfigForEnv(simulated)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cfg})
}

// GetExchangeAnnouncementsHandler GET /exchange/announcements?source=&catalog=&page=
// 交易所公告资讯（币安 CMS + OKX v5 公告聚合）。
func GetExchangeAnnouncementsHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.FetchExchangeAnnouncements(c.Query("source"), c.Query("catalog"), page)})
}

// GetSignalTradePageHandler GET /signals/trades?keyword=&status=&env=&startTime=&endTime=&page=&pageSize=
// OKX 信号自动交易流水分页（面板「模拟交易」页签）。
func GetSignalTradePageHandler(c *gin.Context) {
	query := parseSignalTradeQuery(c)
	page, err := data.NewOKXSignalTradeService().GetSignalTradePage(query)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success",
			"data": models.SignalTradePageData{List: []models.SignalTradeRecord{}}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": *page})
}

// ClearSignalTradeRecordsHandler DELETE /signals/trades?env=
// 清空 OKX 信号自动交易流水；env 为「模拟盘」/「实盘」时只清该环境。
func ClearSignalTradeRecordsHandler(c *gin.Context) {
	if err := data.NewOKXSignalTradeService().ClearSignalTradeRecords(c.Query("env")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}

// GetSignalScanIntervalHandler GET /signals/scan-interval 信号扫描节拍间隔（秒）。
func GetSignalScanIntervalHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data.GetSignalScanIntervalSec()})
}

// SetSignalScanIntervalHandler PUT /signals/scan-interval?sec=
// 设置扫描间隔（1~600 秒）：落库并热更新服务端 SSE 节拍，通知前端重排兜底定时器。
// 下限 1 秒：面板扫描周期提供 1/5/10 秒档位，加密 7x24 行情下高频档位由此放行；
// 1 秒档仅作为唤醒节拍，实际节奏由前端 running 锁与按周期节流保证（一轮扫描跑不满周期时自动跳过多余 tick）。
func SetSignalScanIntervalHandler(c *gin.Context) {
	sec, _ := strconv.Atoi(c.Query("sec"))
	if sec < 1 || sec > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "扫描间隔仅支持 1~600 秒"})
		return
	}
	if err := data.SetSignalScanIntervalSec(sec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": err.Error()})
		return
	}
	// 通知 SSE 总线按新间隔重排节拍（Web 端等价于桌面的 restartSignalMonitorTickCron）
	RestartSignalTickInterval(sec)
	// 通知在线前端重排兜底定时器（桌面走 Wails 事件 signalScanIntervalChanged）
	BroadcastEvent("event: signalScanIntervalChanged\ndata: " + strconv.Itoa(sec) + "\n\n")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": ""})
}

// parseSignalTradeQuery query 参数 → 交易流水查询。
func parseSignalTradeQuery(c *gin.Context) models.SignalTradeQuery {
	return models.SignalTradeQuery{
		Keyword:   c.Query("keyword"),
		Status:    c.Query("status"),
		Env:       c.Query("env"),
		StartTime: parseQueryInt64(c, "startTime"),
		EndTime:   parseQueryInt64(c, "endTime"),
		Page:      int(parseQueryInt64(c, "page")),
		PageSize:  int(parseQueryInt64(c, "pageSize")),
	}
}

// defaultTrue query 参数缺省/空时按 true 处理（OKX 默认模拟盘语义）。
func defaultTrue(v string) bool {
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return b
}
