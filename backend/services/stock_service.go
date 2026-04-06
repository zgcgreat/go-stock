package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// StockService 股票服务
type StockService struct{}

// FollowStock 关注股票
func (s *StockService) FollowStock(stockCode string) string {
	api := data.NewStockDataApi()
	stockInfos, err := api.GetStockCodeRealTimeData(stockCode)
	if err != nil || len(*stockInfos) == 0 {
		logger.SugaredLogger.Error(err)
		return "关注失败"
	}

	processedCode := s.processUSStockCode(stockCode)

	// 检查关注限制
	var count int64
	db.Dao.Model(&data.FollowedStock{}).Where("is_del = ?", 0).Count(&count)
	if count >= 63 {
		return "最多只能关注63只股票"
	}

	// 检查是否已经关注过该股票
	var existingStock data.FollowedStock
	result := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ? AND is_del = ?", processedCode, 0).First(&existingStock)
	if result.Error == nil {
		return "已经关注了"
	}

	// 获取最大排序值
	var maxSort int64
	db.Dao.Model(&data.FollowedStock{}).Raw("SELECT COALESCE(MAX(sort), 0) AS sort FROM followed_stock").Scan(&maxSort)

	stockInfo := (*stockInfos)[0]
	price, _ := convertor.ToFloat(stockInfo.Price)
	newStock := &data.FollowedStock{
		StockCode:          processedCode,
		Name:               stockInfo.Name,
		Price:              price,
		Time:               time.Now(),
		ChangePercent:      0,
		PriceChange:        0,
		Sort:               maxSort + 1,
		AlarmChangePercent: 3,
		AlarmPrice:         price + 1,
	}

	// 创建股票记录
	db.Dao.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("stock_code = ?", processedCode).FirstOrCreate(newStock, &data.FollowedStock{StockCode: processedCode}).Error; err != nil {
			logger.SugaredLogger.Error("创建关注股票失败:", err)
			return err
		}
		return nil
	})

	return "关注成功"
}

// UnFollowStock 取消关注股票
func (s *StockService) UnFollowStock(stockCode string) string {
	processedCode := s.processUSStockCodeForUnfollow(stockCode)

	result := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", strings.ToLower(processedCode)).Delete(&data.FollowedStock{})
	if result.Error != nil {
		logger.SugaredLogger.Error("取消关注股票失败:", result.Error)
		return "取消关注失败"
	}
	return "取消关注成功"
}

// UpdateCostPriceAndVolume 更新股票持仓信息
func (s *StockService) UpdateCostPriceAndVolume(stockCode string, price float64, volume int64) string {
	processedCode := s.processUSStockCodeForUnfollow(stockCode)

	result := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", strings.ToLower(processedCode)).Updates(map[string]interface{}{
		"cost_price": price,
		"volume":     volume,
	})

	if result.Error != nil {
		logger.SugaredLogger.Error("更新持仓信息失败:", result.Error)
		return "设置失败"
	}
	return "设置成功"
}

// UpdateAlarmSettings 更新股票提醒设置
func (s *StockService) UpdateAlarmSettings(stockCode string, changePercent, alarmPrice float64) string {
	processedCode := s.processUSStockCodeForUnfollow(stockCode)

	result := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", strings.ToLower(processedCode)).Updates(map[string]interface{}{
		"alarm_change_percent": changePercent,
		"alarm_price":          alarmPrice,
	})

	if result.Error != nil {
		logger.SugaredLogger.Error("更新提醒设置失败:", result.Error)
		return "设置失败"
	}
	return "设置成功"
}

// UpdateStockSort 更新股票排序
func (s *StockService) UpdateStockSort(stockCode string, newSort int64) {
	processedCode := strings.ToLower(stockCode)

	// 获取当前股票信息
	var currentStock data.FollowedStock
	if err := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", processedCode).First(&currentStock).Error; err != nil {
		logger.SugaredLogger.Error("找不到当前股票:", err)
		return
	}

	oldSort := currentStock.Sort

	// 如果排序值没有变化，直接返回
	if oldSort == newSort {
		return
	}

	// 检查新排序位置是否被占用
	var count int64
	if err := db.Dao.Model(&data.FollowedStock{}).Where("sort = ? AND stock_code != ?", newSort, processedCode).Count(&count).Error; err != nil {
		logger.SugaredLogger.Error("检查新排序位置被占用失败:", err)
		return
	}

	if count == 0 {
		// 新位置未被占用，直接更新当前记录
		if err := db.Dao.Model(&data.FollowedStock{}).
			Where("stock_code = ?", processedCode).
			Update("sort", newSort).Error; err != nil {
			logger.SugaredLogger.Error("更新排序位置失败:", err)
		}
	} else {
		// 新位置已被占用，需要移动其他记录
		if newSort < oldSort {
			// 向前移动：将中间记录向后移动
			if err := db.Dao.Model(&data.FollowedStock{}).
				Where("sort >= ? AND sort < ?", newSort, oldSort).
				UpdateColumn("sort", gorm.Expr("sort + 1")).Error; err != nil {
				logger.SugaredLogger.Error("向前排序更新失败:", err)
			}
		} else {
			// 向后移动：将中间记录向前移动
			if err := db.Dao.Model(&data.FollowedStock{}).
				Where("sort > ? AND sort <= ?", oldSort, newSort).
				UpdateColumn("sort", gorm.Expr("sort - 1")).Error; err != nil {
				logger.SugaredLogger.Error("向后排序更新失败:", err)
			}
		}

		// 更新目标记录的排序
		if err := db.Dao.Model(&data.FollowedStock{}).
			Where("stock_code = ?", processedCode).
			Update("sort", newSort).Error; err != nil {
			logger.SugaredLogger.Error("更新股票排序失败:", err)
		}
	}
}

// UpdateAICronSettings 更新AI定时任务设置
func (s *StockService) UpdateAICronSettings(stockCode string, cron string) {
	processedCode := s.processUSStockCodeForUnfollow(stockCode)

	result := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", strings.ToLower(processedCode)).Update("cron", cron)
	if result.Error != nil {
		logger.SugaredLogger.Error("更新AI定时任务设置失败:", result.Error)
	}
}

// UpdateTradingPrices 更新交易价格信息
func (s *StockService) UpdateTradingPrices(stockCode string, entryPrice, takeProfitPrice, stopLossPrice, costPrice float64) string {
	// 处理不同的股票代码格式
	processedCode := strings.ToUpper(stockCode)
	if strings.HasSuffix(processedCode, ".SZ") {
		processedCode = "sz" + strings.TrimSuffix(processedCode, ".SZ")
	} else if strings.HasSuffix(processedCode, ".SH") {
		processedCode = "sh" + strings.TrimSuffix(processedCode, ".SH")
	} else if strings.HasSuffix(processedCode, ".HK") {
		processedCode = "hk" + strings.TrimSuffix(processedCode, ".HK")
	} else if strings.HasSuffix(processedCode, ".BJ") {
		processedCode = "bj" + strings.TrimSuffix(processedCode, ".BJ")
	} else if strings.HasPrefix(processedCode, "GB_") {
		processedCode = strings.Replace(processedCode, "GB_", "us", 1)
	}

	finalCode := strings.ToLower(processedCode)

	var stock data.FollowedStock
	if err := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", finalCode).First(&stock).Error; err != nil {
		return "股票未关注"
	}

	updates := map[string]interface{}{
		"entry_price":       entryPrice,
		"take_profit_price": takeProfitPrice,
		"stop_loss_price":   stopLossPrice,
		"cost_price":        costPrice,
	}

	result := db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", finalCode).Updates(updates)
	if result.Error != nil {
		return "设置失败"
	}
	if result.RowsAffected == 0 {
		return "设置失败"
	}
	return "设置成功"
}

// GetFollowedList 获取关注列表
func (s *StockService) GetFollowedList(groupId int) *[]data.FollowedStock {
	var result *[]data.FollowedStock

	if groupId == 0 {
		db.Dao.Model(&data.FollowedStock{}).Order("sort ASC, time DESC").Find(&result)
	} else {
		groupAPI := data.NewStockGroupApi(db.Dao)
		infos := groupAPI.GetGroupStockByGroupId(groupId)
		codes := lo.FlatMap(infos, func(info data.GroupStock, idx int) []string {
			return []string{info.StockCode}
		})
		db.Dao.Model(&data.FollowedStock{}).Where("stock_code IN ?", codes).Order("sort ASC, time DESC").Find(&result)
	}

	return result
}

// SearchStocks 搜索股票
func (s *StockService) SearchStocks(keyword string) []data.StockBasic {
	var result []data.StockBasic
	db.Dao.Model(&data.StockBasic{}).Where("name LIKE ? OR ts_code LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Find(&result)

	var result2 []data.IndexBasic
	db.Dao.Model(&data.IndexBasic{}).Where("market IN ?", []string{"SSE", "SZSE"}).Where("name LIKE ? OR ts_code LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Find(&result2)

	var result3 []models.StockInfoHK
	db.Dao.Model(&models.StockInfoHK{}).Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Find(&result3)

	var result4 []models.StockInfoUS
	db.Dao.Model(&models.StockInfoUS{}).Where("name LIKE ? OR code LIKE ? OR e_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").Find(&result4)

	var result5 []models.AllStockInfo
	db.Dao.Model(&models.AllStockInfo{}).Where("secucode LIKE ? OR sec_uri_tynameabbr LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Find(&result5)

	// 创建一个 map 来存储已存在的股票，用于去重
	existingStocks := make(map[string]bool)
	for _, item := range result {
		existingStocks[item.TsCode] = true
	}

	for _, item := range result2 {
		if existingStocks[item.TsCode] {
			continue
		}
		result = append(result, data.StockBasic{
			TsCode:   item.TsCode,
			Name:     item.Name,
			Fullname: item.FullName,
			Symbol:   item.Symbol,
			Market:   item.Market,
			ListDate: item.ListDate,
		})
		existingStocks[item.TsCode] = true
	}

	for _, item := range result3 {
		if existingStocks[item.Code] {
			continue
		}
		result = append(result, data.StockBasic{
			TsCode:   item.Code,
			Name:     item.Name,
			Fullname: item.Name,
			Market:   "HK",
		})
		existingStocks[item.Code] = true
	}

	for _, item := range result4 {
		code := strings.ToLower(strings.Replace(item.Code, "us", "gb_", 1))
		if existingStocks[code] {
			continue
		}
		result = append(result, data.StockBasic{
			TsCode:   code,
			Name:     item.Name,
			Fullname: item.Name,
			Market:   "US",
		})
		existingStocks[code] = true
	}

	for _, item := range result5 {
		if existingStocks[item.SECUCODE] {
			continue
		}
		result = append(result, data.StockBasic{
			TsCode:   item.SECUCODE,
			Name:     item.SECURITYNAMEABBR,
			Fullname: item.SECURITYNAMEABBR,
			Market:   item.MARKET,
		})
		existingStocks[item.SECUCODE] = true
	}

	return result
}

// GetFollowedStockByCode 根据股票代码获取已关注的股票
func (s *StockService) GetFollowedStockByCode(code string) data.FollowedStock {
	var result data.FollowedStock
	db.Dao.Model(&data.FollowedStock{}).Where("stock_code = ?", strings.ToLower(code)).First(&result)
	return result
}

// GetStockRealTimeData 获取股票实时数据
func (s *StockService) GetStockRealTimeData(stockCodes ...string) (*[]data.StockInfo, error) {
	api := data.NewStockDataApi()
	return api.GetStockCodeRealTimeData(stockCodes...)
}

// SaveStockInfo 保存股票信息到数据库
func (s *StockService) SaveStockInfo(ctx context.Context, stockInfo *data.StockInfo) error {
	var count int64
	err := db.Dao.WithContext(ctx).Model(&data.StockInfo{}).Where("code = ?", stockInfo.Code).Count(&count).Error
	if err != nil {
		return err
	}

	if count == 0 {
		return db.Dao.WithContext(ctx).Model(&data.StockInfo{}).Create(stockInfo).Error
	} else {
		return db.Dao.WithContext(ctx).Model(&data.StockInfo{}).Where("code = ?", stockInfo.Code).Updates(stockInfo).Error
	}
}

// 批量保存股票信息
func (s *StockService) SaveBatchStockInfo(ctx context.Context, stockInfos *[]data.StockInfo) error {
	return db.Dao.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, stockInfo := range *stockInfos {
			var count int64
			err := tx.Model(&data.StockInfo{}).Where("code = ?", stockInfo.Code).Count(&count).Error
			if err != nil {
				return err
			}

			if count == 0 {
				if err := tx.Model(&data.StockInfo{}).Create(&stockInfo).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Model(&data.StockInfo{}).Where("code = ?", stockInfo.Code).Updates(&stockInfo).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// processUSStockCode 处理美股股票代码
func (s *StockService) processUSStockCode(stockCode string) string {
	processedCode := stockCode
	if strings.HasPrefix(stockCode, "us") {
		processedCode = strings.Replace(stockCode, "us", "gb_", 1)
	}
	if strings.HasPrefix(stockCode, "US") {
		processedCode = strings.Replace(stockCode, "US", "gb_", 1)
	}
	return strings.ToLower(processedCode)
}

// processUSStockCodeForUnfollow 处理美股股票代码（用于取消关注）
func (s *StockService) processUSStockCodeForUnfollow(stockCode string) string {
	processedCode := stockCode
	if strutil.HasPrefixAny(stockCode, []string{"gb_"}) {
		processedCode = strings.ToUpper(stockCode)
		processedCode = strings.Replace(processedCode, "gb_", "us", 1)
		processedCode = strings.Replace(processedCode, "GB_", "us", 1)
	}
	return processedCode
}

// GetStockDetail 获取股票详细信息
func (s *StockService) GetStockDetail(stockCode string) (*data.StockDetail, error) {
	api := data.NewStockDataApi()

	// 获取实时数据
	realTimeData, err := api.GetStockCodeRealTimeData(stockCode)
	if err != nil || len(*realTimeData) == 0 {
		return nil, fmt.Errorf("无法获取股票实时数据: %v", err)
	}

	stockInfo := (*realTimeData)[0]

	// 获取历史K线数据（最近60天）
	historyData, err := s.getHistoryData(stockCode, 60)
	if err != nil {
		logger.SugaredLogger.Warnf("获取股票历史数据失败 %s: %v", stockCode, err)
	}

	// 获取股票基本信息
	basicInfo, err := s.getStockBasicInfo(stockCode)
	if err != nil {
		logger.SugaredLogger.Warnf("获取股票基本信息失败 %s: %v", stockCode, err)
	}

	// 构造详细信息返回
	detail := &data.StockDetail{
		StockInfo:   stockInfo,
		HistoryData: historyData,
		BasicInfo:   basicInfo,
	}

	return detail, nil
}

// getHistoryData 获取股票历史数据
func (s *StockService) getHistoryData(stockCode string, days int) ([]data.HistoryData, error) {
	// 读取配置，用于创建 EastMoneyKLineApi
	var settings data.Settings
	db.Dao.Model(&data.Settings{}).First(&settings)
	settingConfig := &data.SettingConfig{Settings: &settings}

	api := data.NewEastMoneyKLineApi(settingConfig)
	kLines := api.GetDayKLine(stockCode, days)
	if kLines == nil || len(*kLines) == 0 {
		return nil, nil
	}

	var historyData []data.HistoryData
	for _, k := range *kLines {
		historyData = append(historyData, data.HistoryData{
			Date:          k.Day,
			Open:          k.Open,
			Close:         k.Close,
			High:          k.High,
			Low:           k.Low,
			Volume:        k.Volume,
			Amount:        k.Amount,
			ChangePercent: k.ChangePercent,
		})
	}

	return historyData, nil
}

// getStockBasicInfo 获取股票基本信息
func (s *StockService) getStockBasicInfo(stockCode string) (*data.StockBasic, error) {
	var basicInfo data.StockBasic
	result := db.Dao.Model(&data.StockBasic{}).Where("ts_code = ? OR symbol = ?", stockCode, stockCode).First(&basicInfo)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &basicInfo, nil
}

// GetStockService 获取股票服务实例
func GetStockService() *StockService {
	return &StockService{}
}
