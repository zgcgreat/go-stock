package data

import (
	"fmt"
	"go-stock/backend/db"
	"go-stock/backend/models"
	"time"

	"gorm.io/gorm/clause"
)

type StockChangeHistoryService struct{}

func NewStockChangeHistoryService() *StockChangeHistoryService {
	return &StockChangeHistoryService{}
}

func (s *StockChangeHistoryService) SaveStockChanges(items []StockChangeItem) error {
	if len(items) == 0 {
		return nil
	}

	today := time.Now().Format("2006-01-02")
	var histories []models.StockChangeHistory

	for _, item := range items {
		history := models.StockChangeHistory{
			ChangeTime: item.Time,
			ChangeDate: today,
			StockCode:  item.Code,
			StockName:  item.Name,
			Market:     item.Market,
			ChangeType: item.ChangeType,
			TypeName:   item.TypeName,
			Volume:     item.Volume,
			Price:      item.Price,
			ChangeRate: item.ChangeRate,
			Amount:     item.Amount,
		}
		histories = append(histories, history)
	}

	return db.Dao.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "change_date"}, {Name: "stock_code"}, {Name: "change_time"}},
		DoNothing: true,
	}).CreateInBatches(histories, 100).Error
}

func (s *StockChangeHistoryService) SaveStockChangesWithDedup(items []StockChangeItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}

	today := time.Now().Format("2006-01-02")
	var histories []models.StockChangeHistory
	for _, item := range items {
		history := models.StockChangeHistory{
			ChangeTime: item.Time,
			ChangeDate: today,
			StockCode:  item.Code,
			StockName:  item.Name,
			Market:     item.Market,
			ChangeType: item.ChangeType,
			TypeName:   item.TypeName,
			Volume:     item.Volume,
			Price:      item.Price,
			ChangeRate: item.ChangeRate,
			Amount:     item.Amount,
			Industry:   item.Industry,
			Concept:    item.Concept,
		}
		histories = append(histories, history)
	}

	if len(histories) == 0 {
		return 0, nil
	}

	result := db.Dao.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "change_date"}, {Name: "stock_code"}, {Name: "change_time"}, {Name: "change_type"}, {Name: "price"}, {Name: "change_rate"}, {Name: "amount"}, {Name: "volume"}},
		DoNothing: true,
	}).CreateInBatches(histories, 100)

	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

func (s *StockChangeHistoryService) SaveStockChange(item StockChangeItem) error {
	today := time.Now().Format("2006-01-02")
	history := models.StockChangeHistory{
		ChangeTime: item.Time,
		ChangeDate: today,
		StockCode:  item.Code,
		StockName:  item.Name,
		Market:     item.Market,
		ChangeType: item.ChangeType,
		TypeName:   item.TypeName,
		Volume:     item.Volume,
		Price:      item.Price,
		ChangeRate: item.ChangeRate,
		Amount:     item.Amount,
	}
	return db.Dao.Create(&history).Error
}

func (s *StockChangeHistoryService) GetHistoryList(query models.StockChangeHistoryQuery) (*models.StockChangeHistoryPageData, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 50
	}

	dbQuery := db.Dao.Model(&models.StockChangeHistory{})

	// 股票代码筛选 - 支持模糊匹配
	if query.StockCode != "" {
		dbQuery = dbQuery.Where("stock_code LIKE ?", "%"+query.StockCode+"%")
	}

	// 股票名称筛选 - 支持模糊匹配
	if query.StockName != "" {
		dbQuery = dbQuery.Where("stock_name LIKE ?", "%"+query.StockName+"%")
	}

	// 单个异动类型筛选
	if query.ChangeType > 0 {
		dbQuery = dbQuery.Where("change_type = ?", query.ChangeType)
	}

	// 多个异动类型筛选 - 优先级高于单个类型筛选
	if len(query.ChangeTypes) > 0 {
		dbQuery = dbQuery.Where("change_type IN ?", query.ChangeTypes)
	}

	// 异动类型名称筛选
	if query.TypeName != "" {
		dbQuery = dbQuery.Where("type_name = ?", query.TypeName)
	}

	// 日期范围筛选
	if query.StartDate != "" {
		dbQuery = dbQuery.Where("change_date >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		dbQuery = dbQuery.Where("change_date <= ?", query.EndDate)
	}

	// 时间范围筛选
	if query.StartTime != "" {
		dbQuery = dbQuery.Where("change_time >= ?", query.StartTime)
	}
	if query.EndTime != "" {
		dbQuery = dbQuery.Where("change_time <= ?", query.EndTime)
	}

	// 最小成交量筛选
	if query.MinVolume > 0 {
		dbQuery = dbQuery.Where("volume >= ?", query.MinVolume)
	}

	// 最小金额筛选
	if query.MinAmount > 0 {
		dbQuery = dbQuery.Where("amount >= ?", query.MinAmount)
	}

	// 涨跌幅范围筛选
	if query.MinChangeRate != 0 {
		dbQuery = dbQuery.Where("change_rate >= ?", query.MinChangeRate)
	}
	if query.MaxChangeRate != 0 {
		dbQuery = dbQuery.Where("change_rate <= ?", query.MaxChangeRate)
	}

	// 行业筛选 - 支持模糊匹配
	if query.Industry != "" {
		dbQuery = dbQuery.Where("industry LIKE ?", "%"+query.Industry+"%")
	}

	// 概念筛选 - 支持模糊匹配
	if query.Concept != "" {
		dbQuery = dbQuery.Where("concept LIKE ?", "%"+query.Concept+"%")
	}

	// 计算总数
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// 查询列表数据
	var list []models.StockChangeHistory
	offset := (query.Page - 1) * query.PageSize

	// 按日期和时间倒序排列，这样最新数据在前面
	orderQuery := dbQuery.Order("change_date DESC, change_time DESC")
	if err := orderQuery.Offset(offset).Limit(query.PageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		totalPages++
	}

	// 返回结果
	result := &models.StockChangeHistoryPageData{
		List:       list,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}

	// 添加空数据警告日志
	if len(list) == 0 {
		fmt.Printf("警告: 查询完成但无数据 - 页码:%d, 每页:%d, 总数:%d, 总页:%d, 实际返回:%d, 查询条件: %+v\n",
			query.Page, query.PageSize, total, totalPages, len(list), query)
	} else {
		fmt.Printf("查询完成 - 页码:%d, 每页:%d, 总数:%d, 总页:%d, 实际返回:%d\n",
			query.Page, query.PageSize, total, totalPages, len(list))
	}

	return result, nil
}

func (s *StockChangeHistoryService) DeleteOldData(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	return db.Dao.Where("change_date < ?", cutoffDate).Delete(&models.StockChangeHistory{}).Error
}

func (s *StockChangeHistoryService) GetStockChangeStats(startDate, endDate string) (map[string]interface{}, error) {
	dbQuery := db.Dao.Model(&models.StockChangeHistory{})
	if startDate != "" {
		dbQuery = dbQuery.Where("change_date >= ?", startDate)
	}
	if endDate != "" {
		dbQuery = dbQuery.Where("change_date <= ?", endDate)
	}

	var totalCount int64
	if err := dbQuery.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	type TypeCount struct {
		TypeName string
		Count    int64
	}
	var typeCounts []TypeCount
	if err := db.Dao.Model(&models.StockChangeHistory{}).
		Select("type_name, count(*) as count").
		Where("change_date >= ? AND change_date <= ?", startDate, endDate).
		Group("type_name").
		Order("count DESC").
		Find(&typeCounts).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"totalCount": totalCount,
		"typeCounts": typeCounts,
	}, nil
}
