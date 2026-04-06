package services

import (
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"

	"github.com/duke-git/lancet/v2/lo"
)

// ConfigService 配置服务
type ConfigService struct{}

// GetSettingConfig 获取设置配置
func (s *ConfigService) GetSettingConfig() *data.SettingConfig {
	settingConfig := &data.SettingConfig{}
	settings := &data.Settings{}
	aiConfigs := make([]*data.AIConfig, 0)
	// 处理数据库查询可能返回的空结果
	result := db.Dao.Model(&data.Settings{}).First(settings)
	if settings.OpenAiEnable {
		// 处理AI配置查询可能出现的错误
		result = db.Dao.Model(&data.AIConfig{}).Find(&aiConfigs)
		if result.Error != nil {
			logger.SugaredLogger.Error("查询AI配置失败:", result.Error)
		} else if len(aiConfigs) > 0 {
			lo.ForEach(aiConfigs, func(item *data.AIConfig, index int) {
				if item.TimeOut <= 0 {
					item.TimeOut = 60 * 5
				}
			})
		}
		if settings.CrawlTimeOut <= 0 {
			settings.CrawlTimeOut = 60
		}
		if settings.KDays < 30 {
			settings.KDays = 60
		}
	}
	if settings.BrowserPath == "" {
		settings.BrowserPath, _ = data.CheckBrowser()
	}
	if settings.BrowserPoolSize <= 0 {
		settings.BrowserPoolSize = 1
	}
	settings.EnableFund = false
	settings.EnableAgent = false

	settingConfig.Settings = settings
	settingConfig.AiConfigs = aiConfigs

	return settingConfig
}

// GetConfigService 获取配置服务实例
func GetConfigService() *ConfigService {
	return &ConfigService{}
}