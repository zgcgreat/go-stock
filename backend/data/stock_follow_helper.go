package data

import (
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/mathutil"
)

// AddStockFollowData 为股票数据添加关注信息和计算字段
// 桌面端 app.go 和 Web 端 stock_handler.go 共用，避免重复代码
func AddStockFollowData(follow FollowedStock, stockData *StockInfo) {
	stockData.PrePrice = follow.Price //上次当前价格
	stockData.Sort = follow.Sort
	stockData.CostPrice = follow.CostPrice //成本价
	stockData.CostVolume = follow.Volume   //成本量
	stockData.AlarmChangePercent = follow.AlarmChangePercent
	stockData.AlarmPrice = follow.AlarmPrice
	stockData.Groups = follow.Groups

	//当前价格
	price, _ := convertor.ToFloat(stockData.Price)
	//当前价格为0 时 使用卖一价格作为当前价格
	if price == 0 {
		price, _ = convertor.ToFloat(stockData.A1P)
	}
	//当前价格依然为0 时 使用买一报价作为当前价格
	if price == 0 {
		price, _ = convertor.ToFloat(stockData.B1P)
	}

	//昨日收盘价
	preClosePrice, _ := convertor.ToFloat(stockData.PreClose)

	//当前价格依然为0 时 使用昨日收盘价为当前价格
	if price == 0 {
		price = preClosePrice
	}

	//今日最高价
	highPrice, _ := convertor.ToFloat(stockData.High)
	if highPrice == 0 {
		highPrice, _ = convertor.ToFloat(stockData.Open)
	}

	//今日最低价
	lowPrice, _ := convertor.ToFloat(stockData.Low)
	if lowPrice == 0 {
		lowPrice, _ = convertor.ToFloat(stockData.Open)
	}

	if price > 0 && preClosePrice > 0 {
		stockData.ChangePrice = mathutil.RoundToFloat(price-preClosePrice, 2)
		stockData.ChangePercent = mathutil.RoundToFloat(mathutil.Div(price-preClosePrice, preClosePrice)*100, 3)
	}
	if highPrice > 0 && preClosePrice > 0 {
		stockData.HighRate = mathutil.RoundToFloat(mathutil.Div(highPrice-preClosePrice, preClosePrice)*100, 3)
	}
	if lowPrice > 0 && preClosePrice > 0 {
		stockData.LowRate = mathutil.RoundToFloat(mathutil.Div(lowPrice-preClosePrice, preClosePrice)*100, 3)
	}
	if follow.CostPrice > 0 && follow.Volume > 0 {
		if price > 0 {
			stockData.Profit = mathutil.RoundToFloat(mathutil.Div(price-follow.CostPrice, follow.CostPrice)*100, 3)
			stockData.ProfitAmount = mathutil.RoundToFloat((price-follow.CostPrice)*float64(follow.Volume), 2)
			stockData.ProfitAmountToday = mathutil.RoundToFloat((price-preClosePrice)*float64(follow.Volume), 2)
		} else {
			//未开盘时当前价格为昨日收盘价
			stockData.Profit = mathutil.RoundToFloat(mathutil.Div(preClosePrice-follow.CostPrice, follow.CostPrice)*100, 3)
			stockData.ProfitAmount = mathutil.RoundToFloat((preClosePrice-follow.CostPrice)*float64(follow.Volume), 2)
			// 未开盘时，今日盈亏为 0
			stockData.ProfitAmountToday = 0
		}
	}
}

// GetStockInfoByFollow 根据 FollowedStock 获取股票实时数据并填充关注信息
// 桌面端 app.go 的 getStockInfo 和 Web 端共用
func GetStockInfoByFollow(follow FollowedStock) *StockInfo {
	stockCode := follow.StockCode
	stockDatas, err := NewStockDataApi().GetStockCodeRealTimeData(stockCode)
	if err != nil || stockDatas == nil || len(*stockDatas) == 0 {
		return &StockInfo{}
	}
	stockData := (*stockDatas)[0]
	AddStockFollowData(follow, &stockData)
	return &stockData
}
