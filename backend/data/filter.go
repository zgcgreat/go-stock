package data

import (
	"regexp"
	"strconv"
	"strings"
)

// IndicatorFilterParam 技术指标筛选参数
type IndicatorFilterParam struct {
	// 价格区间
	MinPrice float64 `json:"minPrice"`
	MaxPrice float64 `json:"maxPrice"`

	// 涨跌幅范围
	MinChangePercent float64 `json:"minChangePercent"`
	MaxChangePercent float64 `json:"maxChangePercent"`

	// 成交量相关
	MinVolumeRatio float64 `json:"minVolumeRatio"` // 量比
	MinTurnoverRate float64 `json:"minTurnoverRate"` // 换手率

	// 市值范围
	MinMarketCap float64 `json:"minMarketCap"`
	MaxMarketCap float64 `json:"maxMarketCap"`

	// 技术指标筛选
	HasGoldenForkMACD bool `json:"hasGoldenForkMACD"` // MACD金叉
	HasGoldenForkKDJ  bool `json:"hasGoldenForkKDJ"`  // KDJ金叉

	// 股票代码筛选
	CodePattern string `json:"codePattern"` // 股票代码正则表达式模式

	// 名称筛选
	NameKeywords []string `json:"nameKeywords"` // 股票名称关键词列表
}

// StockIndicatorInterface 定义股票指标的通用接口
type StockIndicatorInterface interface {
	GetPrice() float64
	GetChangePercent() float64
	GetVolumeRatio() float64
	GetTurnoverRate() float64
	GetMarketCap() float64
	GetCode() string
	GetName() string
}

// 定义一个适配器接口，用于处理不同的股票信息结构
type StockInfoAdapter interface {
	GetPrice() float64
	GetChangePercent() float64
	GetVolumeRatio() float64
	GetTurnoverRate() float64
	GetMarketCap() float64
	GetCode() string
	GetName() string
}

// EastMoneyStockInfoAdapter 东财股票信息适配器
type EastMoneyStockInfoAdapter struct {
	Stock *EastMoneyStockInfo
}

func (emsi *EastMoneyStockInfoAdapter) GetPrice() float64 {
	price, err := strconv.ParseFloat(emsi.Stock.Price, 64)
	if err != nil {
		return 0.0
	}
	return price
}

func (emsi *EastMoneyStockInfoAdapter) GetChangePercent() float64 {
	change, err := strconv.ParseFloat(emsi.Stock.ChangePercent, 64)
	if err != nil {
		return 0.0
	}
	return change
}

func (emsi *EastMoneyStockInfoAdapter) GetVolumeRatio() float64 {
	volumeRatio, err := strconv.ParseFloat(emsi.Stock.Volume, 64)
	if err != nil {
		return 0.0
	}
	return volumeRatio
}

func (emsi *EastMoneyStockInfoAdapter) GetTurnoverRate() float64 {
	turnoverRate, err := strconv.ParseFloat(emsi.Stock.Turnover, 64)
	if err != nil {
		return 0.0
	}
	return turnoverRate
}

func (emsi *EastMoneyStockInfoAdapter) GetMarketCap() float64 {
	marketCap, err := strconv.ParseFloat(emsi.Stock.MarketCap, 64)
	if err != nil {
		return 0.0
	}
	return marketCap
}

func (emsi *EastMoneyStockInfoAdapter) GetCode() string {
	return emsi.Stock.Code
}

func (emsi *EastMoneyStockInfoAdapter) GetName() string {
	return emsi.Stock.Name
}

// MatchFilterCondition 检查股票是否符合筛选条件
func MatchFilterCondition(stockInfo *EastMoneyStockInfo, filterParam *IndicatorFilterParam) bool {
	adapter := &EastMoneyStockInfoAdapter{stockInfo}

	// 价格筛选
	if filterParam.MinPrice > 0 && adapter.GetPrice() < filterParam.MinPrice {
		return false
	}
	if filterParam.MaxPrice > 0 && adapter.GetPrice() > filterParam.MaxPrice {
		return false
	}

	// 涨跌幅筛选
	if filterParam.MinChangePercent != 0 && adapter.GetChangePercent() < filterParam.MinChangePercent {
		return false
	}
	if filterParam.MaxChangePercent != 0 && adapter.GetChangePercent() > filterParam.MaxChangePercent {
		return false
	}

	// 量比筛选
	if filterParam.MinVolumeRatio > 0 && adapter.GetVolumeRatio() < filterParam.MinVolumeRatio {
		return false
	}

	// 换手率筛选
	if filterParam.MinTurnoverRate > 0 && adapter.GetTurnoverRate() < filterParam.MinTurnoverRate {
		return false
	}

	// 市值筛选
	if filterParam.MinMarketCap > 0 && adapter.GetMarketCap() < filterParam.MinMarketCap {
		return false
	}
	if filterParam.MaxMarketCap > 0 && adapter.GetMarketCap() > filterParam.MaxMarketCap {
		return false
	}

	// MACD金叉筛选
	// 注意：这里仅为示例，在实际情况中你需要根据股票的具体MACD数据进行判断
	if filterParam.HasGoldenForkMACD {
		// 这里需要具体的MACD计算逻辑，暂时跳过此条件
	}

	// KDJ金叉筛选
	// 注意：这里仅为示例，在实际情况中你需要根据股票的具体KDJ数据进行判断
	if filterParam.HasGoldenForkKDJ {
		// 这里需要具体的KDJ计算逻辑，暂时跳过此条件
	}

	// 代码筛选
	if filterParam.CodePattern != "" {
		matched, err := regexp.MatchString(filterParam.CodePattern, adapter.GetCode())
		if err != nil || !matched {
			return false
		}
	}

	// 名称关键词筛选
	if len(filterParam.NameKeywords) > 0 {
		name := strings.ToLower(adapter.GetName())
		matched := false
		for _, keyword := range filterParam.NameKeywords {
			if strings.Contains(name, strings.ToLower(keyword)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}