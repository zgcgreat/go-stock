package webserver

import (
	"log"
	"sync"
	"time"

	"go-stock/backend/data"
	"go-stock/backend/logger"
)

type Scheduler struct {
	mu        sync.Mutex
	isRunning bool
	stopChan  chan struct{}
}

var (
	scheduler     *Scheduler
	schedulerOnce sync.Once
)

func GetScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		scheduler = &Scheduler{
			stopChan: make(chan struct{}),
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

	go s.runMarketStatisticTask()
	go s.runHotWordsTask()

	log.Println("[Scheduler] Web定时任务调度器已启动")
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}
	close(s.stopChan)
	s.isRunning = false
	log.Println("[Scheduler] Web定时任务调度器已停止")
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
