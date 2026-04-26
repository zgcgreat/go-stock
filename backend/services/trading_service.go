package services

import (
	"errors"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"gorm.io/gorm"
)

// TradingService 交易服务
type TradingService struct{}

// tradingRecordFIFOLot 交易日志先入先出批次
type tradingRecordFIFOLot struct {
	Volume int64
	Price  float64
}

// fifoAvgUnitCost 按先入先出计算卖出数量对应的单位持仓成本（不修改批次）
func (s *TradingService) fifoAvgUnitCost(lots []tradingRecordFIFOLot, sellVol int64) (avg float64, ok bool) {
	if sellVol <= 0 {
		return 0, false
	}

	var totalCost float64
	var totalVol int64

	for _, lot := range lots {
		if totalVol >= sellVol {
			break
		}

		fillVol := lot.Volume
		if totalVol+lot.Volume > sellVol {
			fillVol = sellVol - totalVol
		}

		totalCost += float64(fillVol) * lot.Price
		totalVol += fillVol
	}

	if totalVol < sellVol {
		return 0, false
	}

	return totalCost / float64(totalVol), true
}

// normalizeTradingRecordAPI 将交易日志中的代码转为实时/K 线接口使用的代码
func (s *TradingService) normalizeTradingRecordAPI(stockCode string) string {
	apiCode := stockCode
	if strings.Contains(apiCode, " - ") {
		apiCode = strings.Split(apiCode, " - ")[0]
	}
	apiCode = strings.ToLower(apiCode)
	if strings.HasSuffix(apiCode, ".sh") {
		apiCode = "sh" + strings.TrimSuffix(apiCode, ".sh")
	} else if strings.HasSuffix(apiCode, ".sz") {
		apiCode = "sz" + strings.TrimSuffix(apiCode, ".sz")
	} else if strings.HasSuffix(apiCode, ".bj") {
		apiCode = "bj" + strings.TrimSuffix(apiCode, ".bj")
	} else if strings.HasSuffix(apiCode, ".hk") {
		apiCode = "hk" + strings.TrimSuffix(apiCode, ".hk")
	} else if strings.HasPrefix(apiCode, "gb_") {
		apiCode = strings.Replace(apiCode, "gb_", "us", 1)
	} else if strings.HasPrefix(apiCode, "hk") {
		apiCode = "r_" + apiCode
	}
	return apiCode
}

// resolveTradingRecordClosePrice 按交易日期解析收盘价或现价（无缓存，供写入快照与列表补拉共用）
func (s *TradingService) resolveTradingRecordClosePrice(apiCode string, tradingTime time.Time, fallback float64) float64 {
	if strings.TrimSpace(apiCode) == "" {
		return fallback
	}
	tradingTime = tradingTime.In(time.Local)
	now := time.Now()

	// 非交易时间段的历史数据直接返回当天的收盘价
	if tradingTime.Year() < now.Year() ||
		tradingTime.YearDay() < now.YearDay() ||
		tradingTime.Hour() < 9 ||
		(tradingTime.Hour() >= 11 && tradingTime.Hour() < 14) ||
		tradingTime.Hour() >= 15 {
		// 获取交易日期当天的数据
		// 跳过历史数据获取逻辑，直接返回fallback值
		return fallback
	}

	// 实时数据
	api := data.NewStockDataApi()
	stockDatas, err := api.GetStockCodeRealTimeData(apiCode)
	if err != nil || stockDatas == nil || len(*stockDatas) == 0 {
		logger.SugaredLogger.Debugf("获取实时价格失败: %s, 返回默认价格: %f", err, fallback)
		return fallback
	}

	stock := (*stockDatas)[0]
	price, _ := convertor.ToFloat(stock.Price)
	if price == 0 {
		price, _ = convertor.ToFloat(stock.PreClose)
	}
	if price == 0 {
		price = fallback
	}
	return price
}

// getStockCodeHistoricalData 获取股票历史数据（需要具体实现，这里仅作示意）
func (s *TradingService) getStockCodeHistoricalData(stockCode string, date string) (*[]data.StockInfo, error) {
	// 这里是示意实现，实际应调用历史数据API
	// 暂时返回空数组和错误
	logger.SugaredLogger.Debugf("获取股票历史数据未实现: %s, %s", stockCode, date)
	return &[]data.StockInfo{}, nil

}

// AddTradingRecord 添加交易日志
func (s *TradingService) AddTradingRecord(record data.TradingRecord) (uint, error) {
	record.TradingTime = record.TradingTime.In(time.Local)

	// 检查频繁交易（限定当前用户）
	if record.Direction == "买入" {
		canTrade, msg := s.CheckFrequentTradingWithUser(record.StockCode, record.UserID)
		if !canTrade {
			return 0, errors.New(msg)
		}
	}

	// 自动计算金额（价格 * 数量）
	record.Amount = record.Price * float64(record.Volume)

	// 设置交易时间为当前时间（如果未提供）
	if record.TradingTime.IsZero() {
		record.TradingTime = time.Now()
	}

	s.fillTradingRecordCloseSnapshot(&record)

	// 保存到数据库
	err := db.Dao.Model(&data.TradingRecord{}).Create(&record).Error
	if err != nil {
		logger.SugaredLogger.Errorf("添加交易日志失败: %s", err.Error())
		return 0, err
	}

	return record.ID, nil
}

// GetTradingRecordList 获取交易日志列表（分页、关键词、方向、交易日期范围）
func (s *TradingService) GetTradingRecordList(query data.TradingRecordListQuery) (*data.TradingRecordPageData, error) {
	var records []data.TradingRecord
	q := db.Dao.Model(&data.TradingRecord{})

	page := query.Page
	pageSize := query.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	// 用户隔离：桌面端 user_id=0 不过滤，Web 端传入实际 userID
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}

	// 构建查询条件
	if query.Keyword != "" {
		q = q.Where("stock_code LIKE ? OR stock_name LIKE ?", "%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}
	if query.Direction != "" {
		q = q.Where("direction = ?", query.Direction)
	}
	if query.StartDate != "" {
		startDate, err := time.ParseInLocation("2006-01-02", query.StartDate, time.Local)
		if err == nil {
			q = q.Where("trading_time >= ?", startDate)
		}
	}
	if query.EndDate != "" {
		endDate, err := time.ParseInLocation("2006-01-02", query.EndDate, time.Local)
		if err == nil {
			nextDay := endDate.AddDate(0, 0, 1)
			q = q.Where("trading_time < ?", nextDay)
		}
	}

	// 计算总数
	var total int64
	if err := q.Count(&total).Error; err != nil {
		logger.SugaredLogger.Errorf("查询交易日志总数失败: %s", err.Error())
		return nil, err
	}

	// 分页查询
	if err := q.Order("trading_time DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		logger.SugaredLogger.Errorf("查询交易日志失败: %s", err.Error())
		return nil, err
	}

	// 计算盈亏信息
	needProfitByID := make(map[uint]struct{}, len(records))
	for _, r := range records {
		needProfitByID[r.ID] = struct{}{}
	}

	var allGlobal []data.TradingRecord
	allQ := db.Dao.Model(&data.TradingRecord{})
	if query.UserID > 0 {
		allQ = allQ.Where("user_id = ?", query.UserID)
	}
	if err := allQ.Order("trading_time ASC, id ASC").Find(&allGlobal).Error; err != nil {
		logger.SugaredLogger.Errorf("获取交易日志全局序失败: %s", err.Error())
		return nil, err
	}

	type rowProfit struct {
		id            uint
		closePrice    float64
		profitAmount  float64
		profitPercent float64
	}
	profitByID := make(map[uint]rowProfit, len(records))

	type rowProfitReq struct {
		id          uint
		apiCode     string
		tradingTime time.Time
		fallback    float64
		recorded    float64
		closePrice  float64
	}

	closeCache := make(map[string]float64)

	resolveClose := func(apiCode string, tradingTime time.Time, fallback float64, recorded float64) float64 {
		// 当天或未来日期的记录始终获取实时行情，不使用缓存快照
		tradingDateStr := tradingTime.Format("2006-01-02")
		key := apiCode + "|" + tradingDateStr
		if tradingDateStr == time.Now().Format("2006-01-02") || tradingTime.After(time.Now()) {
			closePrice := s.resolveTradingRecordClosePrice(apiCode, tradingTime, fallback)
			closeCache[key] = closePrice
			return closePrice
		}
		// 历史记录优先使用已保存的快照
		if recorded > 0 {
			return recorded
		}
		if v, ok := closeCache[key]; ok {
			return v
		}
		closePrice := s.resolveTradingRecordClosePrice(apiCode, tradingTime, fallback)
		closeCache[key] = closePrice
		return closePrice
	}

	stockHoldings := make(map[string][]tradingRecordFIFOLot)

	// 异步批量回写收盘价快照
	backfillCh := make(chan rowProfitReq, 16)
	backfillDone := make(chan struct{})
	go func() {
		for bf := range backfillCh {
			if bf.closePrice <= 0 {
				continue
			}
			res := db.Dao.Model(&data.TradingRecord{}).Where("id = ? AND (recorded_close_price IS NULL OR recorded_close_price = 0)", bf.id).
				Update("recorded_close_price", bf.closePrice)
			if res.Error != nil {
				logger.SugaredLogger.Warnf("回写交易记录收盘价快照失败 id=%d: %s", bf.id, res.Error.Error())
			}
		}
		close(backfillDone)
	}()

	for _, r := range allGlobal {
		_, need := needProfitByID[r.ID]
		apiCode := strings.ToLower(r.StockCode)
		tradingDateStr := r.TradingTime.In(time.Local).Format("2006-01-02")
		_ = tradingDateStr

		if r.Direction == "买入" {
			if need {
				closePrice := resolveClose(apiCode, r.TradingTime, r.Price, r.RecordedClosePrice)
				profitAmount := (closePrice - r.Price) * float64(r.Volume)
				profitPercent := 0.0
				if r.Price > 0 {
					profitPercent = (closePrice - r.Price) / r.Price * 100
				}
				profitByID[r.ID] = rowProfit{
					id:            r.ID,
					closePrice:    closePrice,
					profitAmount:  profitAmount,
					profitPercent: profitPercent,
				}
				if r.RecordedClosePrice <= 0 && closePrice > 0 {
					backfillCh <- rowProfitReq{id: r.ID, closePrice: closePrice}
				}
			}
			// 更新持仓
			holdings := stockHoldings[r.StockCode]
			holdings = append(holdings, tradingRecordFIFOLot{
				Volume: r.Volume,
				Price:  r.Price,
			})
			stockHoldings[r.StockCode] = holdings
		} else { // 卖出
			// 计算卖出成本和盈亏
			holdings := stockHoldings[r.StockCode]
			if need {
				unitCost, ok := s.fifoAvgUnitCost(holdings, r.Volume)
				if ok {
					closePrice := resolveClose(apiCode, r.TradingTime, r.Price, r.RecordedClosePrice)
					profit := (closePrice - unitCost) * float64(r.Volume)
					profitPct := 0.0
					if unitCost > 0 {
						profitPct = (closePrice - unitCost) / unitCost * 100
					}
					profitByID[r.ID] = rowProfit{
						id:            r.ID,
						closePrice:    closePrice,
						profitAmount:  profit,
						profitPercent: profitPct,
					}
				} else {
					// 当成本无法计算时，按当前价格相对交易价格的盈亏处理
					closePrice := resolveClose(apiCode, r.TradingTime, r.Price, r.RecordedClosePrice)
					profit := (closePrice - r.Price) * float64(r.Volume)
					profitPct := 0.0
					if r.Price > 0 {
						profitPct = (closePrice - r.Price) / r.Price * 100
					}
					profitByID[r.ID] = rowProfit{
						id:            r.ID,
						closePrice:    closePrice,
						profitAmount:  profit,
						profitPercent: profitPct,
					}
				}
			}
			// 减少持仓数量
			totalVol := r.Volume
			for i := 0; i < len(holdings) && totalVol > 0; i++ {
				lot := &holdings[i]
				if lot.Volume <= 0 {
					continue
				}
				if lot.Volume >= totalVol {
					lot.Volume -= totalVol
					totalVol = 0
				} else {
					totalVol -= lot.Volume
					lot.Volume = 0
				}
			}
			// 移除空批次
			filtered := make([]tradingRecordFIFOLot, 0, len(holdings))
			for _, h := range holdings {
				if h.Volume > 0 {
					filtered = append(filtered, h)
				}
			}
			stockHoldings[r.StockCode] = filtered
		}
	}

	close(backfillCh)
	<-backfillDone

	items := make([]data.TradingRecordItem, 0, len(records))
	for _, r := range records {
		r.Amount = r.Price * float64(r.Volume)
		item := data.TradingRecordItem{TradingRecord: r}
		if p, ok := profitByID[r.ID]; ok {
			item.ClosePrice = p.closePrice
			item.ProfitAmount = p.profitAmount
			item.ProfitPercent = p.profitPercent
		}
		items = append(items, item)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}

	return &data.TradingRecordPageData{
		List:       items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTradingRecordStatistics 获取交易日志统计数据（按用户隔离）
func (s *TradingService) GetTradingRecordStatistics(userID uint) (*data.TradingRecordStatistics, error) {
	type BuyRecord struct {
		Volume int64
		Price  float64
	}

	var records []data.TradingRecord
	q := db.Dao.Model(&data.TradingRecord{})
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Order("trading_time ASC, id ASC").Find(&records).Error
	if err != nil {
		logger.SugaredLogger.Errorf("获取交易日志统计失败: %s", err.Error())
		return nil, err
	}

	// 按股票代码分组的买入数据
	buyMap := make(map[string][]BuyRecord)
	// 按股票代码分组的卖出数据
	sellMap := make(map[string]map[int64]float64) // volume -> price

	var totalBuyAmount float64
	var totalSellAmount float64

	for _, r := range records {
		amount := r.Price * float64(r.Volume)
		if r.Direction == "买入" {
			buyMap[r.StockCode] = append(buyMap[r.StockCode], BuyRecord{
				Volume: r.Volume,
				Price:  r.Price,
			})
			totalBuyAmount += amount
		} else if r.Direction == "卖出" {
			if _, exists := sellMap[r.StockCode]; !exists {
				sellMap[r.StockCode] = make(map[int64]float64)
			}
			sellMap[r.StockCode][r.Volume] = r.Price
			totalSellAmount += amount
		}
	}

	// 计算已售出部分的成本和盈亏
	var costOfSoldShares float64
	var stockCount int64
	var holdingsCost float64
	var holdingsValue float64

	api := data.NewStockDataApi()

	for code, sells := range sellMap {
		buys := buyMap[code]
		for vol, _ := range sells {
			// 对每个卖出记录，按照FIFO原则计算成本
			sellVol := vol
			for i := range buys {
				if sellVol <= 0 {
					break
				}
				buy := &buys[i]
				if buy.Volume <= 0 {
					continue
				}

				// 计算本次能抵扣的买入批次数量
				fillVol := buy.Volume
				if buy.Volume > sellVol {
					fillVol = sellVol
				}

				// 计算成本
				ratio := float64(fillVol) / float64(buy.Volume)
				cost := buy.Price * ratio * float64(fillVol)
				costOfSoldShares += cost

				// 更新买入批次剩余数量
				buy.Volume -= fillVol
				sellVol -= fillVol
			}
		}
	}

	// 计算当前持仓情况
	for code, buys := range buyMap {
		var currentVolume int64
		var currentCost float64

		// 计算剩余持仓数量和成本
		for _, buy := range buys {
			if buy.Volume > 0 {
				currentVolume += buy.Volume
				currentCost += buy.Price * float64(buy.Volume)
			}
		}

		// 如果还有持仓，则加入持仓市值统计
		if currentVolume > 0 {
			stockCount++
			holdingsCost += currentCost

			apiCode := s.normalizeTradingRecordAPI(code)
			stockDatas, err := api.GetStockCodeRealTimeData(apiCode)
			if err == nil && stockDatas != nil && len(*stockDatas) > 0 {
				stock := (*stockDatas)[0]
				price, _ := convertor.ToFloat(stock.Price)
				if price == 0 {
					price, _ = convertor.ToFloat(stock.A1P)
				}
				if price > 0 {
					holdingsValue += price * float64(currentVolume)
				}
			}
		}
	}

	totalProfit := totalSellAmount - costOfSoldShares + (holdingsValue - holdingsCost)
	profitRate := 0.0
	denom := holdingsCost
	if denom <= 0 && costOfSoldShares > 0 {
		denom = costOfSoldShares
	}
	if denom <= 0 && totalBuyAmount > 0 {
		denom = totalBuyAmount
	}
	if denom > 0 {
		profitRate = (totalProfit / denom) * 100
	}

	return &data.TradingRecordStatistics{
		TotalBuyAmount:  totalBuyAmount,
		TotalSellAmount: totalSellAmount,
		TotalProfit:     totalProfit,
		ProfitRate:      profitRate,
		HoldingsAmount:  holdingsCost,
		CurrentValue:    holdingsValue,
		StockCount:      stockCount,
	}, nil
}

// GetTradingRecordById 根据ID获取单个交易日志（用户隔离）
func (s *TradingService) GetTradingRecordById(id uint, userID uint) (*data.TradingRecord, error) {
	var record data.TradingRecord
	q := db.Dao.Model(&data.TradingRecord{}).Where("id = ?", id)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.SugaredLogger.Errorf("获取交易日志失败: %s", err.Error())
		return nil, err
	}
	return &record, nil
}

// UpdateTradingRecord 更新交易日志（用户隔离：禁止修改他人的记录）
func (s *TradingService) UpdateTradingRecord(record data.TradingRecord) error {
	logger.SugaredLogger.Infof("UpdateTradingRecord: %v", record)
	// 自动计算金额（价格 * 数量）
	record.Amount = record.Price * float64(record.Volume)

	record.TradingTime = record.TradingTime.In(time.Local)

	s.fillTradingRecordCloseSnapshot(&record)

	// 用户隔离：禁止修改他人的记录
	q := db.Dao.Model(&data.TradingRecord{}).Where("id = ?", record.ID)
	if record.UserID > 0 {
		q = q.Where("user_id = ?", record.UserID)
	}
	err := q.Updates(&record).Error
	if err != nil {
		logger.SugaredLogger.Errorf("更新交易日志失败: %s", err.Error())
		return err
	}
	// Updates(struct) 会忽略零值字段，收盘价快照单独写入保证落库
	if err := db.Dao.Model(&data.TradingRecord{}).Where("id = ?", record.ID).
		Update("recorded_close_price", record.RecordedClosePrice).Error; err != nil {
		logger.SugaredLogger.Errorf("更新交易日志收盘价快照失败: %s", err.Error())
		return err
	}
	return nil
}

// DeleteTradingRecord 删除交易日志（用户隔离）
func (s *TradingService) DeleteTradingRecord(id uint, userID uint) error {
	q := db.Dao.Model(&data.TradingRecord{}).Where("id = ?", id)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Delete(&data.TradingRecord{}).Error
	if err != nil {
		logger.SugaredLogger.Errorf("删除交易日志失败: %s", err.Error())
		return err
	}
	return nil
}

// CheckFrequentTrading 检查是否频繁交易（桌面端兼容版，不过滤用户）
// 返回值：(是否可以交易, 提示消息)
func (s *TradingService) CheckFrequentTrading(stockCode string) (bool, string) {
	return s.CheckFrequentTradingWithUser(stockCode, 0)
}

// CheckFrequentTradingWithUser 检查是否频繁交易（Web端用户隔离版）
// 返回值：(是否可以交易, 提示消息)
func (s *TradingService) CheckFrequentTradingWithUser(stockCode string, userID uint) (bool, string) {
	// 检查最近24小时内是否有同一只股票的交易日志
	var count int64
	cutoffTime := time.Now().Add(-24 * time.Hour)

	q := db.Dao.Model(&data.TradingRecord{}).Where("stock_code = ? AND direction = ? AND trading_time > ?", stockCode, "买入", cutoffTime)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Count(&count).Error
	if err != nil {
		logger.SugaredLogger.Errorf("检查频繁交易失败: %s", err.Error())
		return true, "检查频繁交易失败，默认允许交易"
	}

	if count > 0 {
		return false, "最近24小时内已对该股票进行过买入操作，为避免频繁交易，建议稍后再操作"
	}

	return true, ""
}

// fillTradingRecordCloseSnapshot 为交易记录填充收盘价快照（历史记录补拉当日收盘价）
func (s *TradingService) fillTradingRecordCloseSnapshot(record *data.TradingRecord) {
	if record.RecordedClosePrice > 0 {
		return
	}
	if record.TradingTime.IsZero() {
		return
	}
	apiCode := s.normalizeTradingRecordAPI(record.StockCode)
	if apiCode == "" {
		return
	}
	closePrice := s.resolveTradingRecordClosePrice(apiCode, record.TradingTime, record.Price)
	if closePrice > 0 {
		record.RecordedClosePrice = closePrice
	}
}

// GetTradingService 获取交易服务实例
func GetTradingService() *TradingService {
	return &TradingService{}
}