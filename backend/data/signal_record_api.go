package data

import (
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SignalRecordService 后台买卖点信号流水的落库与查询。
type SignalRecordService struct{}

func NewSignalRecordService() *SignalRecordService {
	return &SignalRecordService{}
}

// MigrateSignalRecordSchema signal_record 用户隔离迁移（004 批2，幂等，桌面/WEB 两侧启动都必须执行）：
//  1. 补 user_id 列（GORM AutoMigrate 之外对存量库的兜底）；
//  2. 唯一索引重建为 (user_id, code, klt, family, kind, bar_time)——GORM AutoMigrate 不会重建
//     已存在的同名索引，而 SaveSignalRecords 的 ON CONFLICT 冲突目标必须与唯一索引完全一致，
//     存量 5 列索引会让落库直接报错（SQLite: ON CONFLICT does not match）。
//
// 存量数据 user_id=0，桌面语义（0=不过滤）不受影响。
func MigrateSignalRecordSchema() {
	var hasCol int64
	db.Dao.Raw("SELECT COUNT(*) FROM pragma_table_info('signal_record') WHERE name = 'user_id'").Scan(&hasCol)
	if hasCol == 0 {
		if err := db.Dao.Exec("ALTER TABLE signal_record ADD COLUMN user_id INTEGER DEFAULT 0").Error; err != nil {
			logger.SugaredLogger.Errorf("migrate signal_record add user_id error: %s", err.Error())
			return
		}
		logger.SugaredLogger.Info("Migrated: added user_id column to signal_record")
	}

	var hasIdx int64
	db.Dao.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_signal_unique'").Scan(&hasIdx)
	if hasIdx > 0 {
		var withUser int64
		db.Dao.Raw("SELECT COUNT(*) FROM pragma_index_info('idx_signal_unique') WHERE name = 'user_id'").Scan(&withUser)
		if withUser == 0 {
			if err := db.Dao.Exec("DROP INDEX idx_signal_unique").Error; err != nil {
				logger.SugaredLogger.Errorf("migrate signal_record drop legacy index error: %s", err.Error())
				return
			}
			logger.SugaredLogger.Info("Migrated: dropped legacy unique index idx_signal_unique (5-col)")
		}
	}

	if err := db.Dao.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_signal_unique ON signal_record(user_id, code, klt, family, kind, bar_time)").Error; err != nil {
		logger.SugaredLogger.Errorf("migrate signal_record unique index error: %s", err.Error())
	}
}

// SaveSignalRecords 批量落库，按 (user_id,code,klt,family,kind,bar_time) 去重。
// 前端在同一根 K 线上重复上报时静默跳过，不会产生重复流水。
// userID 为 0 表示桌面单用户语义；Web 侧由处理器注入登录用户，请求体中的 UserID 一律被覆盖。
//
// 同一信号可能由模拟盘池与实盘池分别上报（Envs 标注来源池）：去重命中时合并双方环境标注，
// 环境有新增才把该信号重新下发给下游（OKX 自动交易），配合交易流水防重避免同一环境重复下单。
// 返回值是本次真正新增的信号 + 环境被合并的既有信号，供 OKX 信号自动交易等下游消费。
func (s *SignalRecordService) SaveSignalRecords(items []models.SignalRecord, userID uint) ([]models.SignalRecord, error) {
	if len(items) == 0 {
		return nil, nil
	}
	// 用户隔离：落库前一律覆盖为服务端注入的 userID（Web >0 / 桌面 0）
	for i := range items {
		items[i].UserID = userID
	}
	// 预查询批次涉及代码的既有信号（限本用户视角），识别真正新增的记录并支撑环境合并
	codes := make([]string, 0, len(items))
	seenCode := map[string]bool{}
	for _, it := range items {
		if it.Code != "" && !seenCode[it.Code] {
			seenCode[it.Code] = true
			codes = append(codes, it.Code)
		}
	}
	existing := map[string]models.SignalRecord{}
	if len(codes) > 0 {
		var olds []models.SignalRecord
		q := db.Dao.Select("id", "code", "klt", "family", "kind", "bar_time", "envs").Where("code IN ?", codes)
		if userID > 0 {
			q = q.Where("user_id = ?", userID)
		}
		if err := q.Find(&olds).Error; err == nil {
			for _, e := range olds {
				existing[signalUniqueKey(e)] = e
			}
		}
	}

	news := make([]models.SignalRecord, 0, len(items))
	batchSeen := map[string]int{} // key → news 下标（同批重复信号就地合并环境）
	for _, it := range items {
		key := signalUniqueKey(it)
		if old, dup := existing[key]; dup {
			union := mergeEnvTokens(old.Envs, it.Envs)
			if union == old.Envs {
				continue // 环境没有新增，纯重复上报
			}
			if old.ID != 0 {
				db.Dao.Model(&models.SignalRecord{}).Where("id = ?", old.ID).Update("envs", union)
			}
			upd := it
			upd.Envs = union
			news = append(news, upd)
			old.Envs = union
			existing[key] = old
			continue
		}
		if idx, seen := batchSeen[key]; seen {
			news[idx].Envs = mergeEnvTokens(news[idx].Envs, it.Envs)
			continue
		}
		batchSeen[key] = len(news)
		news = append(news, it)
	}
	if len(news) == 0 {
		return nil, nil
	}
	if err := db.Dao.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"}, {Name: "code"}, {Name: "klt"}, {Name: "family"}, {Name: "kind"}, {Name: "bar_time"},
		},
		DoNothing: true,
	}).CreateInBatches(news, 50).Error; err != nil {
		return nil, err
	}
	return news, nil
}

// mergeEnvTokens 合并两个环境标注串（逗号分隔，合法值 sim/live），去重且 sim 恒在前。
func mergeEnvTokens(a, b string) string {
	seen := map[string]bool{}
	out := make([]string, 0, 2)
	for _, v := range []string{a, b} {
		for _, t := range strings.Split(v, ",") {
			t = strings.TrimSpace(t)
			if (t == "sim" || t == "live") && !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	return strings.Join(out, ",")
}

// signalUniqueKey 信号唯一键，与表唯一索引 (user_id,code,klt,family,kind,bar_time) 对应；
// userID 已在调用前注入，此处按记录自身字段拼装即可。
func signalUniqueKey(r models.SignalRecord) string {
	return r.Code + "|" + r.Klt + "|" + r.Family + "|" + r.Kind + "|" + strconv.FormatInt(r.BarTime, 10)
}

// applySourceFilter 按 Source 过滤信号来源：crypto=仅加密（ok:/bn:/bt: 前缀）、stock=仅股票、空=全部。
func applySourceFilter(dbQuery *gorm.DB, source string) *gorm.DB {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "crypto":
		return dbQuery.Where("code LIKE ? OR code LIKE ? OR code LIKE ?", "ok:%", "bn:%", "bt:%")
	case "stock":
		return dbQuery.Where("code NOT LIKE ? AND code NOT LIKE ? AND code NOT LIKE ?", "ok:%", "bn:%", "bt:%")
	}
	return dbQuery
}

// applySignalUserFilter 用户隔离过滤：userID > 0 时仅本用户（Web）；0 不过滤（桌面单用户）。
func applySignalUserFilter(dbQuery *gorm.DB, userID uint) *gorm.DB {
	if userID > 0 {
		return dbQuery.Where("user_id = ?", userID)
	}
	return dbQuery
}

// GetSignalRecordPage 分页查询流水：Keyword 模糊匹配代码/名称，
// StartTime/EndTime 按信号所在 K 线时间（bar_time，Unix 秒）筛选，结果按 K 线时间倒序。
// 分页参数缺失时兜底默认值，避免漏传时把整表拉走。
func (s *SignalRecordService) GetSignalRecordPage(query models.SignalRecordQuery, userID uint) (*models.SignalRecordPageData, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 200 {
		query.PageSize = 20
	}

	dbQuery := db.Dao.Model(&models.SignalRecord{})
	dbQuery = applySignalUserFilter(dbQuery, userID)
	if kw := strings.TrimSpace(query.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbQuery = dbQuery.Where("code LIKE ? OR name LIKE ?", like, like)
	}
	if query.StartTime > 0 {
		dbQuery = dbQuery.Where("bar_time >= ?", query.StartTime)
	}
	if query.EndTime > 0 {
		dbQuery = dbQuery.Where("bar_time <= ?", query.EndTime)
	}
	dbQuery = applySourceFilter(dbQuery, query.Source)
	// 方向过滤：buy=仅买点、sell=仅卖点，空=全部（只认这两个合法值，脏值当全部处理）
	switch strings.ToLower(strings.TrimSpace(query.Kind)) {
	case "buy":
		dbQuery = dbQuery.Where("kind = ?", "buy")
	case "sell":
		dbQuery = dbQuery.Where("kind = ?", "sell")
	}

	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	list := make([]models.SignalRecord, 0)
	if err := dbQuery.Order("bar_time DESC, id DESC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Find(&list).Error; err != nil {
		return nil, err
	}

	totalPages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		totalPages++
	}

	return &models.SignalRecordPageData{
		List:       list,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ClearSignalRecords 清空流水（面板的「清空」按钮）。
// source 语义同 applySourceFilter：股票/OKX 两套监控各自的「清空」只清自己视角的流水，空串=全部。
// userID > 0 时只清该用户的流水（Web 多用户隔离）；0 为桌面语义全表清空。
// 注意：绝不允许在 userID > 0 时走全局删除——那是 004 立项时点名的事故级缺口。
func (s *SignalRecordService) ClearSignalRecords(source string, userID uint) error {
	query := applySourceFilter(
		db.Dao.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(&models.SignalRecord{}),
		source,
	)
	query = applySignalUserFilter(query, userID)
	return query.Delete(&models.SignalRecord{}).Error
}

// signalStatAcc 统计累加器：同一「代码|周期」内的买点排队等卖点来平。
type signalStatAcc struct {
	item    models.SignalStatItem
	pending []signalStatPending // 待平仓的买点，FIFO 逐笔配对
}

// signalStatPending 一个待平仓的信号开仓。
// kind 记录开仓方向（buy=多单 / sell=空单，空单仅加密品种会出现）；
// inRange 标记它是否在统计区间内开仓：区间外开仓的只用来推进配对，不计入「持仓中」。
type signalStatPending struct {
	price   float64
	kind    string
	inRange bool
}

func (a *signalStatAcc) close(ret float64) {
	a.item.Trades++
	if ret > 0 {
		a.item.Wins++
	}
	a.item.Return += ret
}

func (a *signalStatAcc) finish() models.SignalStatItem {
	it := a.item
	for _, p := range a.pending {
		if p.inRange {
			it.Open++
		}
	}
	if it.Trades > 0 {
		it.WinRate = float64(it.Wins) / float64(it.Trades)
	}
	return it
}

// statsLookbackSec 收益统计的回看窗口：区间起点往前多取这么久的历史。
// 取 120 天——远超最长的快捷区间（20 日），又不会把远古未平仓的买点也算成有效持仓。
const statsLookbackSec = 120 * 86400

// GetSignalStats 统计区间内的买卖点信号按「买点开仓、卖点平仓」操作下来的收益与胜率。
//
// 只统计 buysell 族（TEMA 转折是趋势提示，不构成买卖对）；无价格的记录（早期数据）直接跳过。
// 配对在「代码|周期」内进行——同一只票配了多个周期时，各周期的信号各自成笔，不跨周期混算；
// 展示与汇总再按代码合并。未平仓的买点计入 Open（持仓中），不参与胜率。
//
// 加密品种（ok:/bn:/bt: 前缀）与自动交易语义一致：卖点开空、买点平空并开多，
// 每条信号平掉上一笔（方向由开仓信号决定），空单跌了才算盈利；A 股除外，只做多。
//
// 区间起点会往前回看 statsLookbackSec：区间内出现的卖点，其对应买点往往落在区间之外，
// 不回看的话这些平仓笔会被整笔丢弃（区间越短漏得越多，今日区间几乎全漏）。
// 回看进来的买点在区间内的平仓正常计入，区间之前的卖点只用来推进配对队列、不计入统计。
func (s *SignalRecordService) GetSignalStats(query models.SignalStatQuery, userID uint) (*models.SignalStatResult, error) {
	startTime, endTime := query.StartTime, query.EndTime
	dbQuery := db.Dao.Model(&models.SignalRecord{}).Where("family = ?", "buysell")
	dbQuery = applySignalUserFilter(dbQuery, userID)
	dbQuery = applySourceFilter(dbQuery, query.Source)
	if endTime > 0 {
		dbQuery = dbQuery.Where("bar_time <= ?", endTime)
	}
	if startTime > 0 {
		from := startTime - statsLookbackSec
		if from < 0 {
			from = 0
		}
		dbQuery = dbQuery.Where("bar_time >= ?", from)
	}

	list := make([]models.SignalRecord, 0)
	// 必须按时间正序配对：先买后卖
	if err := dbQuery.Order("bar_time ASC, id ASC").Find(&list).Error; err != nil {
		return nil, err
	}

	byKey := make(map[string]*signalStatAcc)
	keys := make([]string, 0)

	for i := range list {
		r := list[i]
		if r.Price == nil {
			continue
		}
		inRange := startTime <= 0 || r.BarTime >= startTime
		key := r.Code + "|" + r.Klt
		acc, ok := byKey[key]
		if !ok {
			acc = &signalStatAcc{item: models.SignalStatItem{Code: r.Code, Name: r.Name}}
			byKey[key] = acc
			keys = append(keys, key)
		}
		// 加密品种（ok:/bn:/bt: 前缀）按自动交易的真实语义统计：
		// 卖点开空、买点平空开多——每条信号平掉上一笔并开新仓，方向由开仓信号决定，
		// 空单盈亏 = (开空价-平仓价)/开空价（跌了才赚）。A 股除外，维持只做多语义。
		if _, isCrypto := okxSignalCodeInstInput(r.Code); isCrypto {
			if len(acc.pending) > 0 {
				prev := acc.pending[0]
				acc.pending = acc.pending[1:]
				if inRange {
					if prev.kind == "sell" {
						acc.close((prev.price - *r.Price) / prev.price) // 空单
					} else {
						acc.close((*r.Price - prev.price) / prev.price) // 多单
					}
				}
			}
			acc.pending = append(acc.pending, signalStatPending{price: *r.Price, kind: r.Kind, inRange: inRange})
			continue
		}
		if r.Kind == "buy" {
			acc.pending = append(acc.pending, signalStatPending{price: *r.Price, kind: r.Kind, inRange: inRange})
			continue
		}
		if len(acc.pending) == 0 {
			// 卖点没有对应买点（回看窗口内也没有），不成笔
			continue
		}
		buy := acc.pending[0]
		acc.pending = acc.pending[1:]
		if !inRange {
			// 区间之前的卖点：只推进配对队列，不计入本次统计
			continue
		}
		acc.close((*r.Price - buy.price) / buy.price)
	}

	// 按代码合并（同一只票的多个周期合并成一条）
	byCode := make(map[string]models.SignalStatItem)
	codes := make([]string, 0)
	for _, key := range keys {
		it := byKey[key].finish()
		m, ok := byCode[it.Code]
		if !ok {
			m = models.SignalStatItem{Code: it.Code, Name: it.Name}
			codes = append(codes, it.Code)
		}
		if m.Name == "" {
			m.Name = it.Name
		}
		m.Trades += it.Trades
		m.Wins += it.Wins
		m.Return += it.Return
		m.Open += it.Open
		byCode[it.Code] = m
	}

	stocks := make([]models.SignalStatItem, 0, len(codes))
	// 整体统计由按股票的汇总求和得出：与明细同一份来源，不会出现两边对不上的情况
	overall := models.SignalStatItem{}
	for _, code := range codes {
		it := byCode[code]
		if it.Trades > 0 {
			it.WinRate = float64(it.Wins) / float64(it.Trades)
		}
		stocks = append(stocks, it)
		overall.Trades += it.Trades
		overall.Wins += it.Wins
		overall.Return += it.Return
		overall.Open += it.Open
	}
	if overall.Trades > 0 {
		overall.WinRate = float64(overall.Wins) / float64(overall.Trades)
	}
	// 收益高的排前面，便于直接看哪只票贡献最大
	sort.SliceStable(stocks, func(i, j int) bool {
		if stocks[i].Return != stocks[j].Return {
			return stocks[i].Return > stocks[j].Return
		}
		return stocks[i].Code < stocks[j].Code
	})

	return &models.SignalStatResult{Overall: overall, Stocks: stocks}, nil
}
