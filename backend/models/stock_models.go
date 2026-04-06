package models

import (
	"time"
	"gorm.io/plugin/soft_delete"
)

// AIRecommendStocks AI推荐股票增强版 - 支持多租户
type AIRecommendStocks struct {
	Base      // 使用多租户基础模型
	Code            string      `json:"code" gorm:"index;comment:股票代码"`
	Name            string      `json:"name" gorm:"index;comment:股票名称"`
	Date            string      `json:"date" gorm:"index;comment:推荐日期"`
	Reason          string      `json:"reason" gorm:"type:text;comment:推荐理由"`
	ModelUsed       string      `json:"modelUsed" gorm:"comment:使用的AI模型"`
	Confidence      float64     `json:"confidence" gorm:"comment:置信度"`
	ExpectedReturn  float64     `json:"expectedReturn" gorm:"comment:预期收益率"`
	Strategy        string      `json:"strategy" gorm:"comment:策略名称"`
	RiskLevel       string      `json:"riskLevel" gorm:"comment:风险等级"`
	AnalysisReport  string      `json:"analysisReport" gorm:"type:text;comment:分析报告"`
	PriceAtRecommendation float64 `json:"priceAtRecommendation" gorm:"comment:推荐时价格"`
	CurrentPrice    float64     `json:"currentPrice" gorm:"comment:当前价格"`
	ReturnRate      float64     `json:"returnRate" gorm:"comment:收益率"`
}

func (AIRecommendStocks) TableName() string {
	return "ai_recommend_stocks"
}

// AIRecommendStocksHistory AI推荐股票历史增强版 - 支持多租户
type AIRecommendStocksHistory struct {
	Base      // 使用多租户基础模型
	RecommendId     uint        `json:"recommendId" gorm:"index;comment:关联的推荐ID"`
	Code            string      `json:"code" gorm:"index;comment:股票代码"`
	Name            string      `json:"name" gorm:"index;comment:股票名称"`
	Date            string      `json:"date" gorm:"index;comment:历史日期"`
	Price           float64     `json:"price" gorm:"comment:当日价格"`
	ChangePercent   float64     `json:"changePercent" gorm:"comment:涨跌幅"`
	ChangeAmount    float64     `json:"changeAmount" gorm:"comment:涨跌金额"`
	Volume          int64       `json:"volume" gorm:"comment:成交量"`
	TurnoverRate    float64     `json:"turnoverRate" gorm:"comment:换手率"`
	MarketCap       float64     `json:"marketCap" gorm:"comment:市值"`
	PE              float64     `json:"pe" gorm:"comment:市盈率"`
	PB              float64     `json:"pb" gorm:"comment:市净率"`
	Note            string      `json:"note" gorm:"type:text;comment:备注"`
}

func (AIRecommendStocksHistory) TableName() string {
	return "ai_recommend_stocks_history"
}

// AIRecommendStocksSummary 推荐股票统计摘要增强版 - 支持多租户
type AIRecommendStocksSummary struct {
	Base      // 使用多租户基础模型
	StartDate       string  `json:"startDate" gorm:"index;comment:开始日期"`
	EndDate         string  `json:"endDate" gorm:"index;comment:结束日期"`
	TotalRecommendations int `json:"totalRecommendations" gorm:"comment:总推荐数"`
	SuccessfulRecommendations int `json:"successfulRecommendations" gorm:"comment:成功推荐数(正收益)"`
	AverageReturn   float64 `json:"averageReturn" gorm:"comment:平均收益率"`
	BestReturn      float64 `json:"bestReturn" gorm:"comment:最佳收益率"`
	WorstReturn     float64 `json:"worstReturn" gorm:"comment:最差收益率"`
	WinRate         float64 `json:"winRate" gorm:"comment:胜率"`
	TotalProfitLoss float64 `json:"totalProfitLoss" gorm:"comment:总盈亏"`
	ModelUsed       string  `json:"modelUsed" gorm:"comment:使用的AI模型"`
	Strategy        string  `json:"strategy" gorm:"comment:策略名称"`
	Notes           string  `json:"notes" gorm:"type:text;comment:统计说明"`
}

func (AIRecommendStocksSummary) TableName() string {
	return "ai_recommend_stocks_summary"
}

// StockTradeRecord 股票交易记录增强版 - 支持多租户
type StockTradeRecord struct {
	Base      // 使用多租户基础模型
	ID              uint      `gorm:"primaryKey" json:"id"`
	TsCode          string    `json:"ts_code" gorm:"index;comment:股票代码"`
	Symbol          string    `json:"symbol" gorm:"index;comment:股票代码"`
	Name            string    `json:"name" gorm:"index;comment:股票名称"`
	Code            string    `json:"code" gorm:"index;comment:精简代码"`
	TradeType       string    `json:"trade_type" gorm:"index;comment:交易类型:buy/sell"`
	Direction       string    `json:"direction" gorm:"index;comment:方向:buy/sell"`
	Volume          float64   `json:"volume" gorm:"comment:交易数量"`
	Price           float64   `json:"price" gorm:"comment:交易价格"`
	Amount          float64   `json:"amount" gorm:"comment:交易金额"`
	Commission      float64   `json:"commission" gorm:"comment:佣金费用"`
	Taxes           float64   `json:"taxes" gorm:"comment:税费"`
	TradingCosts    float64   `json:"trading_costs" gorm:"comment:交易成本"`
	TradingTime     time.Time `json:"trading_time" gorm:"index;comment:交易时间"`
	CloseTime       *time.Time `json:"close_time" gorm:"comment:平仓时间"`
	ClosePrice      *float64   `json:"close_price" gorm:"comment:平仓价格"`
	CloseVolume     *float64   `json:"close_volume" gorm:"comment:平仓数量"`
	Profit          *float64   `json:"profit" gorm:"comment:盈亏金额"`
	ProfitRate      *float64   `json:"profit_rate" gorm:"comment:盈亏率"`
	Reason          string    `json:"reason" gorm:"type:text;comment:交易理由"`
	Mindset         string    `json:"mindset" gorm:"type:text;comment:交易思路/策略"`
	RecordedClosePrice *float64 `json:"recordedClosePrice" gorm:"comment:记录的收盘价"`
}

func (StockTradeRecord) TableName() string {
	return "stock_trade_records"
}

// StockPool 股票池增强版 - 支持多租户
type StockPool struct {
	Base      // 使用多租户基础模型
	TsCode    string `json:"ts_code" gorm:"index;comment:TS代码"`
	Symbol    string `json:"symbol" gorm:"index;comment:股票代码"`
	Name      string `json:"name" gorm:"index;comment:股票名称"`
	Area      string `json:"area" gorm:"index;comment:地域"`
	Industry  string `json:"industry" gorm:"index;comment:行业"`
	Market    string `json:"market" gorm:"index;comment:市场类型"`
	ListDate  string `json:"list_date" gorm:"index;comment:上市日期"`
	IsDel     soft_delete.DeletedAt `gorm:"softDelete:flag"`
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uint `gorm:"index;comment:用户ID"`

	// 额外的池相关字段
	PoolName      string  `json:"pool_name" gorm:"index;comment:股票池名称"`
	PoolCategory  string  `json:"pool_category" gorm:"index;comment:股票池类别"`
	Weight        float64 `json:"weight" gorm:"comment:权重"`
	LastRank      int     `json:"last_rank" gorm:"comment:上次排名"`
	RankingScore  float64 `json:"ranking_score" gorm:"comment:排名分数"`
	InPoolDate    time.Time `json:"in_pool_date" gorm:"comment:入池时间"`
	OutPoolDate   *time.Time `json:"out_pool_date" gorm:"comment:出池时间"`
	IsActive      bool    `json:"is_active" gorm:"default:true;comment:是否活跃"`
	Priority      int     `json:"priority" gorm:"comment:优先级"`
	Notes         string  `json:"notes" gorm:"type:text;comment:备注"`
}

func (StockPool) TableName() string {
	return "stock_pools"
}