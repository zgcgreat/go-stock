package agent

import (
	"errors"
	"fmt"
	"strings"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"gorm.io/gorm"
)

// UserProfileSnapshot 返回画像页面所需的统一状态，避免前端多次 IPC 调用产生状态不同步。
type UserProfileSnapshot struct {
	Content     string `json:"content"`
	Enabled     bool   `json:"enabled"`
	UpdatedAt   string `json:"updatedAt"`
	KnownFields int    `json:"knownFields"`
	TotalFields int    `json:"totalFields"`
}

// user_profile_api.go — 用户画像 Wails API（P1）。
//
// 遵循 KnowledgeBaseApi / CronTaskApi 薄包装模式，把 UserProfileLearner 的能力
// 暴露给前端"Agent 对我的了解"页面：预览 / 覆盖 / 重新学习 / 清空。
//
// 用法（app.go 中）：
//
//	agent.NewUserProfileApi().GetUserProfile()
//
// 前端 wailsjs 绑定：
//	GetUserProfile / SaveUserProfile / RelearnUserProfile / ClearUserProfile

// UserProfileApi 用户画像 API（Wails 绑定）
type UserProfileApi struct{}

// NewUserProfileApi 构造用户画像 API 实例
func NewUserProfileApi() *UserProfileApi {
	return &UserProfileApi{}
}

// GetUserProfile 读取当前画像内容（未生成返回空）。
func (a *UserProfileApi) GetUserProfile() string {
	return NewUserProfileLearner().Get()
}

func (a *UserProfileApi) GetUserProfileUpdatedAt() string {
	return NewUserProfileLearner().UpdatedAt()
}

// GetUserProfileSnapshot 一次读取画像内容、启用状态、更新时间和完整度。
func (a *UserProfileApi) GetUserProfileSnapshot() *UserProfileSnapshot {
	content := NewUserProfileLearner().Get()
	total := len(userProfileFieldLabels)
	known := 0
	for _, label := range userProfileFieldLabels {
		for _, line := range strings.Split(content, "\n") {
			prefix := "- " + label + "："
			if strings.HasPrefix(strings.TrimSpace(line), prefix) {
				value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), prefix))
				if value != "" && value != "未明确" && value != "无" {
					known++
				}
				break
			}
		}
	}
	return &UserProfileSnapshot{Content: content, Enabled: IsUserProfileEnabled(), UpdatedAt: NewUserProfileLearner().UpdatedAt(), KnownFields: known, TotalFields: total}
}

// GetUserProfileEnabled 返回用户画像是否会注入 Agent。
func (a *UserProfileApi) GetUserProfileEnabled() bool {
	return IsUserProfileEnabled()
}

// SetUserProfileEnabled 设置用户画像是否注入 Agent。
func (a *UserProfileApi) SetUserProfileEnabled(enabled bool) error {
	return SetUserProfileEnabled(enabled)
}

// SaveUserProfile 手动覆盖画像。
func (a *UserProfileApi) SaveUserProfile(content string) error {
	return NewUserProfileLearner().Save(content)
}

// RelearnUserProfile 一键重新学习画像，返回新画像内容。
func (a *UserProfileApi) RelearnUserProfile() (string, error) {
	return NewUserProfileLearner().Relearn()
}

// ClearUserProfile 清空用户画像（含重置增强标记）。
func (a *UserProfileApi) ClearUserProfile() error {
	return NewUserProfileLearner().Clear()
}

// GetProfileLearnAiConfigId 读取画像学习模型设置（0=自动模式）。桌面端全局语义。
func (a *UserProfileApi) GetProfileLearnAiConfigId() int {
	cfg := data.GetSettingConfig()
	if cfg == nil {
		return 0
	}
	return cfg.ProfileLearnAiConfigId
}

// GetProfileLearnAiConfigIdByUserID 读取指定用户的画像学习模型设置（0=自动模式）。
// Web 端用户隔离：用户行不存在时 GetSettingConfigByUserID 会按系统默认配置回退，全新库不再报错。
func (a *UserProfileApi) GetProfileLearnAiConfigIdByUserID(userID uint) int {
	cfg := data.GetSettingConfigByUserID(userID)
	if cfg == nil || cfg.Settings == nil {
		return 0
	}
	return cfg.Settings.ProfileLearnAiConfigId
}

// SetProfileLearnAiConfigId 设置画像学习模型（aiConfigId 传 0 恢复自动模式）。
// 保存后立即生效（learner 每次学习时重新读取设置）。
// 直接更新 DB 单字段（UpdateConfig 已不再写该字段，避免主设置页零值覆盖问题）。
// 全新库 Settings 表为空时自动创建默认行（此前 First 报 record not found → 500）。
func (a *UserProfileApi) SetProfileLearnAiConfigId(aiConfigId int) error {
	if aiConfigId < 0 {
		aiConfigId = 0
	}
	var s data.Settings
	err := db.Dao.First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s = data.Settings{ProfileLearnAiConfigId: aiConfigId}
		if e := db.Dao.Create(&s).Error; e != nil {
			return fmt.Errorf("初始化 settings 失败: %w", e)
		}
		logger.SugaredLogger.Infof("画像学习模型已设置为 aiConfigId=%d（新建 settings 行）", aiConfigId)
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取 settings 失败: %w", err)
	}
	if err := db.Dao.Model(&data.Settings{}).Where("id=?", s.ID).
		Update("profile_learn_ai_config_id", aiConfigId).Error; err != nil {
		return fmt.Errorf("保存画像学习模型失败: %w", err)
	}
	logger.SugaredLogger.Infof("画像学习模型已设置为 aiConfigId=%d", aiConfigId)
	return nil
}

// SetProfileLearnAiConfigIdByUserID 设置指定用户的画像学习模型（0=恢复自动模式）。
// Web 端用户隔离：按 user_id 定位行；行不存在时先从系统默认配置（user_id=0）复制落库，
// 连默认行都没有（全新库）则创建仅含目标字段的最小行——修复全新部署 record not found 500。
func (a *UserProfileApi) SetProfileLearnAiConfigIdByUserID(userID uint, aiConfigId int) error {
	if aiConfigId < 0 {
		aiConfigId = 0
	}
	var s data.Settings
	err := db.Dao.Where("user_id = ?", userID).First(&s).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		def := data.Settings{}
		if e := db.Dao.Where("user_id = ?", 0).First(&def).Error; e == nil {
			// 复制系统默认配置为新用户行（对齐 GetSettingConfigByUserID 的复制语义）
			def.ID = 0
			def.UserID = userID
			def.ProfileLearnAiConfigId = aiConfigId
			if e := db.Dao.Create(&def).Error; e != nil {
				return fmt.Errorf("初始化用户设置失败: %w", e)
			}
			logger.SugaredLogger.Infof("画像学习模型已设置为 aiConfigId=%d (userID=%d，自默认配置复制)", aiConfigId, userID)
			return nil
		}
		// 全新库：无任何 settings 行，创建最小行
		s = data.Settings{UserID: userID, ProfileLearnAiConfigId: aiConfigId}
		if e := db.Dao.Create(&s).Error; e != nil {
			return fmt.Errorf("初始化用户设置失败: %w", e)
		}
		logger.SugaredLogger.Infof("画像学习模型已设置为 aiConfigId=%d (userID=%d，新建设置行)", aiConfigId, userID)
		return nil
	case err != nil:
		return fmt.Errorf("读取 settings 失败: %w", err)
	}
	if err := db.Dao.Model(&data.Settings{}).Where("id=?", s.ID).
		Update("profile_learn_ai_config_id", aiConfigId).Error; err != nil {
		return fmt.Errorf("保存画像学习模型失败: %w", err)
	}
	logger.SugaredLogger.Infof("画像学习模型已设置为 aiConfigId=%d (userID=%d)", aiConfigId, userID)
	return nil
}
