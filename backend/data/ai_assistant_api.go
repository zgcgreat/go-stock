package data

import (
	"encoding/json"
	"go-stock/backend/db"
	"go-stock/backend/models"
	"regexp"
	"strings"
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

// 从会话消息 JSON 前缀中抓取首条 user 消息的 content 作为标题。
// 列表接口只取 substr(messages,1,4000) 前缀，JSON 可能被截断而无法整体反序列化，
// 故用字符串扫描 + 正则提取，再对捕获值做 JSON 反转义。
var (
	sessionUserContentRegex = regexp.MustCompile(`"role"\s*:\s*"user"[\s\S]*?"content"\s*:\s*"((?:[^"\\]|\\.)*)"`)
	sessionAnyContentRegex  = regexp.MustCompile(`"content"\s*:\s*"((?:[^"\\]|\\.)*)"`)
)

func extractSessionTitle(head string) string {
	if head == "" {
		return ""
	}
	raw := ""
	if m := sessionUserContentRegex.FindStringSubmatch(head); len(m) > 1 {
		raw = m[1]
	} else if m := sessionAnyContentRegex.FindStringSubmatch(head); len(m) > 1 {
		raw = m[1]
	}
	if raw == "" {
		return ""
	}
	title := raw
	var decoded string
	if err := json.Unmarshal([]byte(`"`+raw+`"`), &decoded); err == nil {
		title = decoded
	}
	title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
	runes := []rune(title)
	if len(runes) > 40 {
		title = string(runes[:40])
	}
	return title
}

// aiAssistantSessionRow 会话列表查询的中间结果（仅取消息前缀用于生成标题）。
type aiAssistantSessionRow struct {
	SessionId string
	UpdatedAt time.Time
	Head      string
}

// ListAiAssistantSessions 返回会话的元信息（sessionId/标题/更新时间），按更新时间倒序（用户隔离）。
// userID > 0 时仅返回该用户的会话；keyword 非空时对会话正文做模糊匹配（标题由正文首条用户消息派生，
// 故可一并命中），用于搜索历史会话；为空时仅返回最近 limit 条（limit<=0 时默认 10 条），
// 避免一次性加载全部会话。仅取 messages 前 4000 字符生成标题，避免全量反序列化超大消息列。
func ListAiAssistantSessions(userID uint, keyword string, limit int) ([]models.AiAssistantSessionMeta, error) {
	if limit <= 0 {
		limit = 10
	}
	query := db.Dao.Model(&models.AiAssistantSession{}).
		Select("session_id, updated_at, substr(messages, 1, 4000) as head").
		Order("updated_at DESC").
		Limit(limit)
	// 用户隔离
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		// 转义 LIKE 通配符，避免用户输入 % / _ 被当作模式匹配
		escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(kw)
		query = query.Where("messages LIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}
	var rows []aiAssistantSessionRow
	err := query.Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	list := make([]models.AiAssistantSessionMeta, 0, len(rows))
	for _, r := range rows {
		list = append(list, models.AiAssistantSessionMeta{
			SessionId: r.SessionId,
			Title:     extractSessionTitle(r.Head),
			UpdatedAt: r.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return list, nil
}

// DeleteAiAssistantSession 删除指定会话（用户隔离），并清理其对话记忆（chat_memory）。
func DeleteAiAssistantSession(sessionId string, userID uint) error {
	if sessionId == "" {
		return nil
	}
	q := db.Dao.Model(&models.AiAssistantSession{}).Where("session_id = ?", sessionId)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Delete(&models.AiAssistantSession{}).Error; err != nil {
		return err
	}
	return db.ClearChatMemory(sessionId)
}
