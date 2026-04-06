package services

import (
	"go-stock/backend/data"
	"go-stock/backend/models"
)

// IndicatorService 技术指标服务
type IndicatorService struct{}

// FilterStocksByIndicators 根据技术指标筛选股票
func (s *IndicatorService) FilterStocksByIndicators(filterParams *data.IndicatorFilterParam) ([]*models.StockInfo, error) {
	stockInfos := &[]*models.StockInfo{}
	api := data.NewEastMoneyAPI()

	// 获取所有股票信息
	allStocks, err := api.GetAllStockInfo()
	if err != nil {
		return nil, err
	}

	// 由于EastMoneyStockInfo和models.StockInfo结构不同，我们需要进行适配
	for _, emStock := range allStocks {
		// 使用东财适配器进行筛选
		if data.MatchFilterCondition(emStock, filterParams) {
			// 转换为models.StockInfo类型
			modelsStock := &models.StockInfo{
				SECUCODE:           emStock.Code,
				SECURITYNAMEABBR:   emStock.Name,
				NEWPRICE:     emStock.Price,
				CHANGERATE:    emStock.ChangePercent,
				VOLUMERATIO:      emStock.Volume,
				TURNOVERRATE:     emStock.Turnover,
			}
			*stockInfos = append(*stockInfos, modelsStock)
		}
	}

	return *stockInfos, nil
}

// GetIndicatorService 获取指标服务实例
func GetIndicatorService() *IndicatorService {
	return &IndicatorService{}
}