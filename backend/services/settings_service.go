package services

import (
	"encoding/json"
	"errors"
	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"

	"github.com/samber/lo"
	"gorm.io/gorm"
)

// SettingsService 配置服务
type SettingsService struct{}

// UpdateConfig 更新配置
func (s *SettingsService) UpdateConfig(settingConfig *data.SettingConfig) string {
	count := int64(0)
	db.Dao.Model(&data.Settings{}).Count(&count)
	if count > 0 {
		db.Dao.Model(&data.Settings{}).Where("id=?", settingConfig.ID).Updates(map[string]any{
			"local_push_enable":          settingConfig.LocalPushEnable,
			"ding_push_enable":           settingConfig.DingPushEnable,
			"ding_robot":                 settingConfig.DingRobot,
			"update_basic_info_on_start": settingConfig.UpdateBasicInfoOnStart,
			"refresh_interval":           settingConfig.RefreshInterval,
			"open_ai_enable":             settingConfig.OpenAiEnable,
			"tushare_token":              settingConfig.TushareToken,
			"prompt":                     settingConfig.Prompt,
			"check_update":               settingConfig.CheckUpdate,
			"question_template":          settingConfig.QuestionTemplate,
			"crawl_time_out":             settingConfig.CrawlTimeOut,
			"k_days":                     settingConfig.KDays,
			"enable_danmu":               settingConfig.EnableDanmu,
			"browser_path":               settingConfig.BrowserPath,
			"enable_news":                settingConfig.EnableNews,
			"dark_theme":                 settingConfig.DarkTheme,
			"enable_fund":                settingConfig.EnableFund,
			"enable_push_news":           settingConfig.EnablePushNews,
			"enable_only_push_red_news":  settingConfig.EnableOnlyPushRedNews,
			"sponsor_code":               settingConfig.SponsorCode,
			"http_proxy":                 settingConfig.HttpProxy,
			"http_proxy_enabled":         settingConfig.HttpProxyEnabled,
			"enable_agent":               settingConfig.EnableAgent,
			"qgqp_b_id":                  settingConfig.QgqpBId,
			"window_width":               settingConfig.WindowWidth,
			"window_height":              settingConfig.WindowHeight,
		})

		// 更新AiConfig
		err := s.updateAiConfigs(settingConfig.AiConfigs)
		if err != nil {
			logger.SugaredLogger.Errorf("更新AI模型服务配置失败: %v", err)
			return "更新AI模型服务配置失败: " + err.Error()
		}
	} else {
		//logger.SugaredLogger.Infof("未找到配置，创建默认配置")
		// 创建主配置
		result := db.Dao.Model(&data.Settings{}).Create(&data.Settings{})
		if result.Error != nil {
			logger.SugaredLogger.Error("创建配置失败:", result.Error)
			return "创建配置失败: " + result.Error.Error()
		}
	}
	return "保存成功！"
}

// updateAiConfigs 更新AI配置
func (s *SettingsService) updateAiConfigs(aiConfigs []*data.AIConfig) error {
	if len(aiConfigs) == 0 {
		err := db.Dao.Exec("DELETE FROM ai_config").Error
		if err != nil {
			return err
		}
		return db.Dao.Exec("DELETE FROM sqlite_sequence WHERE name='ai_config'").Error
	}
	// 仅收集大于 0 的 ID，用于识别已存在的配置；
	// ID<=0 视为"新配置"，强制走插入逻辑，避免多个 ID 为 0 的配置互相覆盖。
	var ids []uint
	lo.ForEach(aiConfigs, func(item *data.AIConfig, index int) {
		if item.ID > 0 {
			ids = append(ids, item.ID)
		}
	})
	var existAiConfigs []*data.AIConfig
	err := db.Dao.Model(&data.AIConfig{}).Select("id").Where("id in (?) ", ids).Find(&existAiConfigs).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	idMap := make(map[uint]bool)
	lo.ForEach(existAiConfigs, func(item *data.AIConfig, index int) {
		idMap[item.ID] = true
	})
	var addAiConfigs []*data.AIConfig
	var notDeleteIds []uint
	var e error
	lo.ForEach(aiConfigs, func(item *data.AIConfig, index int) {
		if e != nil {
			return
		}
		// ID<=0 一律视为新配置，走插入逻辑；否则根据是否已存在决定更新或新增
		if item.ID <= 0 || !idMap[item.ID] {
			addAiConfigs = append(addAiConfigs, item)
		} else {
			notDeleteIds = append(notDeleteIds, item.ID)
			e = db.Dao.Model(&data.AIConfig{}).Where("id=?", item.ID).Updates(map[string]interface{}{
				"name":               item.Name,
				"base_url":           item.BaseUrl,
				"api_key":            item.ApiKey,
				"model_name":         item.ModelName,
				"max_tokens":         item.MaxTokens,
				"temperature":        item.Temperature,
				"time_out":           item.TimeOut,
				"http_proxy":         item.HttpProxy,
				"http_proxy_enabled": item.HttpProxyEnabled,
				"session_id":         item.SessionId,
			}).Error
			if e != nil {
				return
			}
		}
	})
	if e != nil {
		return e
	}
	// 删除旧的配置
	if len(notDeleteIds) > 0 {
		err = db.Dao.Exec("DELETE FROM ai_config WHERE id NOT IN ?", notDeleteIds).Error
		if err != nil {
			return err
		}
	}
	//logger.SugaredLogger.Infof("更新aiConfigs +%d", len(addAiConfigs))
	// 批量新增的配置
	err = db.Dao.CreateInBatches(addAiConfigs, len(addAiConfigs)).Error
	return err
}

// GetSettingConfig 获取设置配置
func (s *SettingsService) GetSettingConfig() *data.SettingConfig {
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

// ExportConfig 导出配置
func (s *SettingsService) ExportConfig(config *data.SettingConfig) string {
	d, _ := json.MarshalIndent(config, "", "    ")
	return string(d)
}

// GetSettingsService 获取配置服务实例
func GetSettingsService() *SettingsService {
	return &SettingsService{}
}