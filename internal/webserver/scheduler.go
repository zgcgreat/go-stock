package webserver

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/robfig/cron/v3"

	"go-stock/backend/agent"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"
)

type Scheduler struct {
	mu         sync.Mutex
	isRunning  bool
	stopChan   chan struct{}
	cron       *cron.Cron
	cronEntrys map[uint]cron.EntryID // key: CronTask.ID
}

var (
	scheduler     *Scheduler
	schedulerOnce sync.Once
)

func GetScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		c := cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(cron.DefaultLogger)))
		scheduler = &Scheduler{
			stopChan:   make(chan struct{}),
			cron:       c,
			cronEntrys: make(map[uint]cron.EntryID),
		}
	})
	return scheduler
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	// 启动时检查股票基础数据是否为空，若为空则触发初始化（一次性）
	go s.initStockDataIfEmpty()

	go s.runMarketStatisticTask()
	go s.runBKFundFlowTask()
	go s.runHotWordsTask()

	// 启动用户自定义定时任务调度器
	s.initCronTasks()
	s.cron.Start()

	log.Println("[Scheduler] Web定时任务调度器已启动")
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	// 停止 cron 调度器
	if s.cron != nil {
		s.cron.Stop()
	}

	close(s.stopChan)
	s.isRunning = false
	log.Println("[Scheduler] Web定时任务调度器已停止")
}

// initCronTasks 从数据库加载所有启用的用户定时任务并注册到 cron 调度器
func (s *Scheduler) initCronTasks() {
	cronApi := agent.NewCronTaskApi()
	tasks := cronApi.GetAll() // 只返回 enable=true 的任务
	for _, t := range tasks {
		taskCopy := t
		entryID, err := s.cron.AddFunc(taskCopy.CronExpr, func() {
			err := agent.NewCronTaskApi().ExecuteTask(context.Background(), &taskCopy)
			if err != nil {
				logger.SugaredLogger.Errorf("[Scheduler] 执行定时任务失败：%v (%s)", err, taskCopy.Name)
			}
		})
		if err != nil {
			logger.SugaredLogger.Errorf("[Scheduler] 注册定时任务失败：%v (%s cron=%s)", err, taskCopy.Name, taskCopy.CronExpr)
			continue
		}
		s.cronEntrys[taskCopy.ID] = entryID
		logger.SugaredLogger.Infof("[Scheduler] 注册定时任务: %s (cron=%s entryID=%d)", taskCopy.Name, taskCopy.CronExpr, entryID)
	}
}

// AddCronTask 动态添加定时任务到调度器（创建/启用任务时调用）
func (s *Scheduler) AddCronTask(task *models.CronTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已有旧的 entry，先移除
	if oldID, exists := s.cronEntrys[task.ID]; exists {
		s.cron.Remove(oldID)
		delete(s.cronEntrys, task.ID)
	}

	if !task.Enable {
		// 任务已禁用，只移除不添加
		logger.SugaredLogger.Infof("[Scheduler] 任务已禁用，移除调度: %s (ID=%d)", task.Name, task.ID)
		return nil
	}

	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		err := agent.NewCronTaskApi().ExecuteTask(context.Background(), task)
		if err != nil {
			logger.SugaredLogger.Errorf("[Scheduler] 执行定时任务失败：%v (%s)", err, task.Name)
		}
	})
	if err != nil {
		return err
	}
	s.cronEntrys[task.ID] = entryID
	logger.SugaredLogger.Infof("[Scheduler] 添加定时任务: %s (cron=%s entryID=%d)", task.Name, task.CronExpr, entryID)
	return nil
}

// RemoveCronTask 从调度器移除定时任务（删除/禁用任务时调用）
func (s *Scheduler) RemoveCronTask(taskID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.cronEntrys[taskID]; exists {
		s.cron.Remove(entryID)
		delete(s.cronEntrys, taskID)
		logger.SugaredLogger.Infof("[Scheduler] 移除定时任务调度 (ID=%d entryID=%d)", taskID, entryID)
	}
}

func (s *Scheduler) runMarketStatisticTask() {
	s.fetchMarketStatistic()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			log.Println("[Scheduler] 市场统计数据采集任务已停止")
			return
		case <-ticker.C:
			if isTradingTime() {
				s.fetchMarketStatistic()
			}
		}
	}
}

func (s *Scheduler) fetchMarketStatistic() {
	defer func() {
		if r := recover(); r != nil {
			logger.SugaredLogger.Errorf("[Scheduler] 获取市场统计数据失败: %v", r)
		}
	}()

	err := data.NewMarketStatisticApi().FetchAndSave()
	if err != nil {
		logger.SugaredLogger.Errorf("[Scheduler] 获取市场统计数据失败: %v", err)
		return
	}
	logger.SugaredLogger.Debugf("[Scheduler] 市场统计数据采集成功")
}

func (s *Scheduler) runBKFundFlowTask() {
	s.fetchBKFundFlow()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			log.Println("[Scheduler] 板块资金流向采集任务已停止")
			return
		case <-ticker.C:
			if isTradingTime() {
				s.fetchBKFundFlow()
			}
		}
	}
}

func (s *Scheduler) fetchBKFundFlow() {
	defer func() {
		if r := recover(); r != nil {
			logger.SugaredLogger.Errorf("[Scheduler] 获取板块资金流向失败: %v", r)
		}
	}()

	count, err := data.NewBKFundFlowApi().FetchAndSave()
	if err != nil {
		logger.SugaredLogger.Errorf("[Scheduler] 获取板块资金流向失败: %v", err)
		return
	}
	logger.SugaredLogger.Debugf("[Scheduler] 板块资金流向采集成功，保存 %d 条", count)
}

func (s *Scheduler) runHotWordsTask() {
	s.fetchHotWords()

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			log.Println("[Scheduler] 热词采集任务已停止")
			return
		case <-ticker.C:
			if isTradingTime() {
				s.fetchHotWords()
			}
		}
	}
}

func (s *Scheduler) fetchHotWords() {
	defer func() {
		if r := recover(); r != nil {
			logger.SugaredLogger.Errorf("[Scheduler] 获取热词失败: %v", r)
		}
	}()

	newsApi := data.NewMarketNewsApi()
	newsApi.TelegraphList(30)
	newsApi.GetSinaNews(30)
	logger.SugaredLogger.Debugf("[Scheduler] 热词采集成功")
}

// initStockDataIfEmpty 检查股票基础数据是否为空，若为空则触发初始化（一次性）
func (s *Scheduler) initStockDataIfEmpty() {
	defer func() {
		if r := recover(); r != nil {
			logger.SugaredLogger.Errorf("[Scheduler] 股票数据初始化失败: %v", r)
		}
	}()

	// 检查 StockBasic 和 AllStockInfo 是否为空
	var count int64
	db.Dao.Model(&models.AllStockInfo{}).Count(&count)
	if count > 0 {
		log.Println("[Scheduler] 股票基础数据已存在，跳过初始化")
		return
	}

	log.Println("[Scheduler] 检测到股票基础数据为空，开始初始化...")

	// 1. 从东方财富爬取全量 A 股信息写入 AllStockInfo（无需 Token）
	syncAllStockInfo()

	// 2. 从远程 JSON 拉取 StockBasic（无需 Token）
	fetchStockBasic()

	// 3. 重新计数确认
	db.Dao.Model(&models.AllStockInfo{}).Count(&count)
	log.Printf("[Scheduler] 股票数据初始化完成，AllStockInfo 共 %d 条", count)
}

// syncAllStockInfo 从东方财富爬取全量 A 股信息（复用 app.go 逻辑）
func syncAllStockInfo() {
	db.Dao.Unscoped().Model(&models.AllStockInfo{}).Where("1=1").Delete(&models.AllStockInfo{})
	for page := 1; page <= 2; page++ {
		res := data.NewStockDataApi().GetAllStocks(page, 3000, "", models.TechnicalIndicators{})
		if res == nil || len(res.Result.Data) == 0 {
			continue
		}
		var datas []models.AllStockInfo
		for _, d := range res.Result.Data {
			datas = append(datas, d.ToAllStockInfo())
		}
		if err := db.Dao.CreateInBatches(&datas, 1000).Error; err != nil {
			logger.SugaredLogger.Errorf("[Scheduler] 同步 AllStockInfo page=%d 失败: %v", page, err)
		}
	}
}

// fetchStockBasic 从远程 JSON 拉取 StockBasic（复用 app.go CheckStockBaseInfo 逻辑）
func fetchStockBasic() {
	stockBasics := &[]data.StockBasic{}
	_, err := resty.New().R().
		SetHeader("user", "go-stock").
		SetResult(stockBasics).
		Get("http://8.134.249.145:18080/go-stock/stock_basic.json")
	if err != nil {
		logger.SugaredLogger.Errorf("[Scheduler] 拉取 stock_basic.json 失败: %v", err)
		return
	}
	if len(*stockBasics) == 0 {
		logger.SugaredLogger.Warn("[Scheduler] stock_basic.json 返回为空")
		return
	}
	if err := db.Dao.Exec("DELETE FROM stock_basics").Error; err != nil {
		logger.SugaredLogger.Errorf("[Scheduler] 清空 stock_basics 失败: %v", err)
	}
	if err := db.Dao.CreateInBatches(stockBasics, 400).Error; err != nil {
		logger.SugaredLogger.Errorf("[Scheduler] 写入 stock_basics 失败: %v", err)
	}
	log.Printf("[Scheduler] StockBasic 写入 %d 条", len(*stockBasics))
}

func isTradingTime() bool {
	now := time.Now()
	weekday := now.Weekday()

	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	hour := now.Hour()
	minute := now.Minute()
	currentTime := hour*100 + minute

	morningStart := 915
	morningEnd := 1130
	afternoonStart := 1257
	afternoonEnd := 1500

	return (currentTime >= morningStart && currentTime <= morningEnd) ||
		(currentTime >= afternoonStart && currentTime <= afternoonEnd)
}
