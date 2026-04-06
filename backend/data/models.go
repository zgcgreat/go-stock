package data

import (
	"gorm.io/gorm"
)
type StockDetail struct {
	StockInfo   StockInfo      `json:"stockInfo"`   // 实时数据
	HistoryData []HistoryData  `json:"historyData"` // 历史数据
	BasicInfo   *StockBasic    `json:"basicInfo"`   // 基本信息
}

// HistoryData 历史数据
type HistoryData struct {
	Date          string  `json:"date"`          // 日期
	Open          string  `json:"open"`          // 开盘价
	Close         string  `json:"close"`         // 收盘价
	High          string  `json:"high"`          // 最高价
	Low           string  `json:"low"`           // 最低价
	Volume        string  `json:"volume"`        // 成交量
	Amount        string  `json:"amount"`        // 成交额
	ChangePercent string  `json:"changePercent"` // 涨跌幅
}

// StockBasic 股票基本信息结构
type StockBasic struct {
	gorm.Model

	TsCode      string  `json:"ts_code" gorm:"column:ts_code;index"`  // 股票代码
	Name      string  `json:"name" gorm:"column:name;index"`     // 股票名称
	Fullname  string  `json:"fullname" gorm:"column:fullname"`   // 股票全称
	Symbol    string  `json:"symbol" gorm:"column:symbol"`       // 交易代码
	Market    string  `json:"market" gorm:"column:market;index"` // 交易市场
	ListDate  string  `json:"listDate" gorm:"column:list_date"`  // 上市日期
	Industry  string  `json:"industry" gorm:"column:industry"`   // 行业

	// 针对不同市场的特殊字段
	Exchange  string `json:"exchange" gorm:"column:exchange"`   // 交易所（主要用于美股）
	Type      string `json:"type" gorm:"column:type"`          // 类型（主要用于美股）
	Area      string `json:"area" gorm:"column:area"`          // 地域
	BKName    string `json:"bk_name" gorm:"column:bk_name"`    // 板块名称
	BKCode    string `json:"bk_code" gorm:"column:bk_code"`    // 板块代码
	MarketType string `json:"marketType" gorm:"column:market_type"` // 市场类型
}