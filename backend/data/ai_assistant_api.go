package data

import (
	"encoding/json"
	"go-stock/backend/db"
	"go-stock/backend/models"
	"time"
)

const maxSavedMessages = 65535 * 10000

// GetAiAssistantSession 获取指定 sessionId 的会话消息列表（用户隔离）
// userID > 0 时：必须匹配 sessionId 和 userId，否则返回空
// userID = 0 时：只匹配 sessionId（桌面端兼容）
func GetAiAssistantSession(sessionId string, userID uint) (*models.AiAssistantSessionResp, error) {
	var row models.AiAssistantSession
	var err error
	q := db.Dao.Model(&models.AiAssistantSession{})
	if sessionId != "" {
		q = q.Where("session_id = ?", sessionId)
	}
	// 用户隔离
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if sessionId != "" {
		err = q.First(&row).Error
	} else {
		err = q.Order("updated_at DESC").First(&row).Error
	}
	resp := &models.AiAssistantSessionResp{
		Messages:  []models.AiAssistantMessage{},
		SessionId: row.SessionId,
	}
	if err != nil {
		return resp, nil
	}
	if row.Messages == "" {
		return resp, nil
	}
	var list []models.AiAssistantMessage
	if err := json.Unmarshal([]byte(row.Messages), &list); err != nil {
		return resp, nil
	}
	resp.Messages = list
	return resp, nil
}

// SaveAiAssistantSession 保存会话消息到数据库（用户隔离）
// userID > 0 时：创建/更新时关联 userId，禁止跨用户覆盖
// userID = 0 时：桌面端兼容，不限制
func SaveAiAssistantSession(sessionId string, userID uint, messages []models.AiAssistantMessage) error {
	if len(messages) == 0 {
		return nil
	}
	toSave := messages
	if len(toSave) > maxSavedMessages {
		toSave = toSave[len(toSave)-maxSavedMessages:]
	}
	raw, err := json.Marshal(toSave)
	if err != nil {
		return err
	}
	payload := string(raw)

	// 查询现有会话
	q := db.Dao.Model(&models.AiAssistantSession{}).Where("session_id = ?", sessionId)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	var existing models.AiAssistantSession
	err = q.First(&existing).Error
	if err == nil {
		// 存在则更新
		return db.Dao.Model(&models.AiAssistantSession{}).Where("session_id = ?", sessionId).Updates(map[string]interface{}{
			"messages":   payload,
			"updated_at": time.Now(),
		}).Error
	}
	// 不存在则创建新记录，关联 userID
	session := models.AiAssistantSession{
		SessionId: sessionId,
		UserID:    userID,
		Messages:  payload,
	}
	return db.Dao.Create(&session).Error
}
