package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// EastMoneyAPI 是用于获取东财网数据的结构
type EastMoneyAPI struct {
	Client *resty.Client
}

// NewEastMoneyAPI 创建东财网API实例
func NewEastMoneyAPI() *EastMoneyAPI {
	client := resty.New().
		SetTimeout(30*time.Second).
		SetHeader("Accept", "*/*").
		SetHeader("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8").
		SetHeader("Cache-Control", "no-cache").
		SetHeader("Pragma", "no-cache").
		SetHeader("Referer", "https://quote.eastmoney.com/").
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return &EastMoneyAPI{
		Client: client,
	}
}

// StockInfo 用于表示从东方财富获取的股票信息
type EastMoneyStockInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Price string `json:"price"`
	ChangePercent string `json:"changePercent"`
	ChangeAmount string `json:"changeAmount"`
	Volume string `json:"volume"`
	Turnover string `json:"turnover"`
	Amplitude string `json:"amplitude"`
	High52Week string `json:"high52week"`
	Low52Week string `json:"low52week"`
	MarketCap string `json:"marketCap"`
	PE string `json:"pe"`
	PB string `json:"pb"`
	TotalShares string `json:"totalShares"`
}

// GetAllStockInfo 获取所有股票信息
func (api *EastMoneyAPI) GetAllStockInfo() ([]*EastMoneyStockInfo, error) {
	// 东方财富的股票数据API地址
	// 这里只是一个模拟示例，实际地址需要根据东财的API而定
	reqURL := "https://push2.eastmoney.com/api/qt/clist/get"

	// 构建参数
	params := map[string]string{
		"pn": "1",           // 页码
		"pz": "1000",        // 每页数量
		"po": "1",           // 排序方式
		"np": "1",           // 是否分页
		"fltt": "2",         // 刷选时间
		"invt": "2",         // 机构类型
		"ut": "b2884a393a717e00dc7bd7aa3dd6ccfe", // 用户令牌
	}

	resp, err := api.Client.R().
		SetQueryParams(params).
		SetHeader("Referer", "https://quote.eastmoney.com/").
		Get(reqURL)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode())
	}

	// 解析响应数据
	var responseMap map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &responseMap); err != nil {
		return nil, err
	}

	// 检查是否有错误
	if responseMap["error_code"] != nil && responseMap["error_code"].(float64) != 0 {
		return nil, fmt.Errorf("API返回错误: %v", responseMap["error_msg"])
	}

	// 提取股票信息
	var stocks []*EastMoneyStockInfo

	if data, ok := responseMap["data"].(map[string]interface{}); ok {
		if diff, ok := data["diff"].([]interface{}); ok {
			for _, item := range diff {
				if stockMap, ok := item.(map[string]interface{}); ok {
					stock := &EastMoneyStockInfo{
						Code: fmt.Sprintf("%v", stockMap["f12"]),
						Name: fmt.Sprintf("%v", stockMap["f14"]),
						Price: fmt.Sprintf("%v", stockMap["f2"]),
						ChangePercent: fmt.Sprintf("%v", stockMap["f3"]),
						ChangeAmount: fmt.Sprintf("%v", stockMap["f4"]),
						Volume: fmt.Sprintf("%v", stockMap["f5"]),
						Turnover: fmt.Sprintf("%v", stockMap["f6"]),
						Amplitude: fmt.Sprintf("%v", stockMap["f7"]),
						MarketCap: fmt.Sprintf("%v", stockMap["f204"]),
					}

					stocks = append(stocks, stock)
				}
			}
		}
	}

	return stocks, nil
}

// GetStockInfo 获取个股信息
func (api *EastMoneyAPI) GetStockInfo(stockCode string) (*EastMoneyStockInfo, error) {
	reqURL := "https://push2.eastmoney.com/api/qt/stock/get"

	params := map[string]string{
		"ut": "b2884a393a717e00dc7bd7aa3dd6ccfe",
		"fltt": "2",
		"invt": "2",
		"fields": "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f12,f13,f14,f15,f16,f17,f18,f20,f21,f23,f24,f25,f22,f11,f62,f128,f136,f152",
		"secid": api.convertSecID(stockCode),
	}

	resp, err := api.Client.R().
		SetQueryParams(params).
		Get(reqURL)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode())
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &responseMap); err != nil {
		return nil, err
	}

	if responseMap["error_code"] != nil && responseMap["error_code"].(float64) != 0 {
		return nil, fmt.Errorf("API返回错误: %v", responseMap["error_msg"])
	}

	if data, ok := responseMap["data"].(map[string]interface{}); ok {
		stock := &EastMoneyStockInfo{
			Code: stockCode,
			Name: fmt.Sprintf("%v", data["f14"]),
			Price: fmt.Sprintf("%v", data["f2"]),
			ChangePercent: fmt.Sprintf("%v", data["f3"]),
			ChangeAmount: fmt.Sprintf("%v", data["f4"]),
			Volume: fmt.Sprintf("%v", data["f5"]),
			Turnover: fmt.Sprintf("%v", data["f6"]),
		}

		return stock, nil
	}

	return nil, fmt.Errorf("未能获取股票数据")
}

// convertSecID 转换股票代码为东方财富标准格式
func (api *EastMoneyAPI) convertSecID(stockCode string) string {
	stockCode = strings.ToUpper(stockCode)

	if strings.HasPrefix(stockCode, "SH") || strings.HasPrefix(stockCode, "SZ") {
		return stockCode
	}

	if strings.HasPrefix(stockCode, "0") || strings.HasPrefix(stockCode, "3") {
		return fmt.Sprintf("SZ%s", stockCode)
	} else if strings.HasPrefix(stockCode, "6") {
		return fmt.Sprintf("SH%s", stockCode)
	} else if strings.HasPrefix(stockCode, "8") {
		return fmt.Sprintf("BJ%s", stockCode)
	}

	// 如果是以其他方式开头的尝试添加"GB_"前缀表示美股
	if strings.HasPrefix(stockCode, "GB_") {
		return strings.Replace(stockCode, "GB_", "", 1)
	}

	// 对于美股代码，通常直接返回
	return stockCode
}

// GetStockHistory 获取股票历史数据
func (api *EastMoneyAPI) GetStockHistory(stockCode string, period string, count int) ([]map[string]interface{}, error) {
	reqURL := "https://push2his.eastmoney.com/api/qt/stock/kline/get"

	params := map[string]string{
		"secid": api.convertSecID(stockCode),
		"ut": "fa5fd1943c7b386f172d6893dbfba10b",
		"fields1": "f1,f2,f3,f4,f5",
		"fields2": "f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61",
		"klt": api.periodToKLT(period),
		"fh": "1",
		"rnd": fmt.Sprintf("%d", time.Now().Unix()),
		"end": "20500101",
		"lmt": fmt.Sprintf("%d", count),
	}

	resp, err := api.Client.R().
		SetQueryParams(params).
		Get(reqURL)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode())
	}

	var responseMap map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &responseMap); err != nil {
		return nil, err
	}

	if responseMap["error_code"] != nil && responseMap["error_code"].(float64) != 0 {
		return nil, fmt.Errorf("API返回错误: %v", responseMap["error_msg"])
	}

	var result []map[string]interface{}

	if data, ok := responseMap["data"].(map[string]interface{}); ok {
		if kLines, ok := data["klines"].([]interface{}); ok {
			for _, kLine := range kLines {
				lineStr := fmt.Sprintf("%v", kLine)
				parts := strings.Split(lineStr, ",")

				if len(parts) >= 8 {
					kData := map[string]interface{}{
						"date": parts[0],
						"open": parts[1],
						"close": parts[2],
						"high": parts[3],
						"low": parts[4],
						"volume": parts[5],
						"amount": parts[6],
						"changePercent": parts[7],
					}
					result = append(result, kData)
				}
			}
		}
	}

	return result, nil
}

// periodToKLT 将周期转换为东方财富标准周期代码
func (api *EastMoneyAPI) periodToKLT(period string) string {
	switch strings.ToLower(period) {
	case "daily", "day":
		return "101" // 日K
	case "weekly", "week":
		return "102" // 周K
	case "monthly", "month":
		return "103" // 月K
	case "1min":
		return "1" // 1分钟K
	case "5min":
		return "5" // 5分钟K
	case "15min":
		return "15" // 15分钟K
	case "30min":
		return "30" // 30分钟K
	case "60min":
		return "60" // 60分钟K
	default:
		return "101" // 默认日K
	}
}