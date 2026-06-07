package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PlazaSyncService 提示词广场数据同步服务
type PlazaSyncService struct {
	apiBase string
	token   string
	client  *http.Client
}

func NewPlazaSyncService(apiBase string) *PlazaSyncService {
	if apiBase == "" {
		apiBase = "http://go-stock.sparkmemory.top:1918/api"
	}
	return &PlazaSyncService{
		apiBase: apiBase,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// --- 外部API原始数据结构 ---

type plazaAPIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type plazaLoginData struct {
	Token string       `json:"token"`
	User  plazaUserRaw `json:"user"`
}

type plazaUserRaw struct {
	ID       uint   `json:"id"`
	Nickname string `json:"nickname"`
	Username string `json:"username"`
	VipLevel int    `json:"vipLevel"`
}

type plazaPromptListData struct {
	List     []plazaPromptRaw `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

type plazaPromptRaw struct {
	ID             uint         `json:"id"`
	Title          string       `json:"title"`
	Content        string       `json:"content"`
	Description    string       `json:"description"`
	Summary        string       `json:"summary"`
	Category       string       `json:"category"`
	Tags           string       `json:"tags"`
	IsPublic       bool         `json:"isPublic"`
	VipOnly        bool         `json:"vipOnly"`
	NeedVip        bool         `json:"needVip"`
	UserID         uint         `json:"userId"`
	User           plazaUserRaw `json:"user"`
	ViewsCount     int          `json:"viewsCount"`
	LikesCount     int          `json:"likesCount"`
	FavoritesCount int          `json:"favoritesCount"`
	DownloadsCount int          `json:"downloadsCount"`
	CommentsCount  int          `json:"commentsCount"`
	HotScore       float64      `json:"hotScore"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      string       `json:"updatedAt"`
}

type plazaQuestionListData struct {
	List     []plazaQuestionRaw `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type plazaQuestionRaw struct {
	ID           uint         `json:"id"`
	Title        string       `json:"title"`
	Content      string       `json:"content"`
	IsResolved   bool         `json:"isResolved"`
	UserID       uint         `json:"userId"`
	User         plazaUserRaw `json:"user"`
	AnswersCount int          `json:"answersCount"`
	CreatedAt    string       `json:"createdAt"`
	UpdatedAt    string       `json:"updatedAt"`
}

type plazaQuestionDetailData struct {
	Question plazaQuestionRaw `json:"question"`
	Answers  []plazaAnswerRaw `json:"answers"`
}

type plazaAnswerRaw struct {
	ID         uint         `json:"id"`
	Content    string       `json:"content"`
	IsAccepted bool         `json:"isAccepted"`
	IsLiked    bool         `json:"isLiked"`
	LikesCount int          `json:"likesCount"`
	UserID     uint         `json:"userId"`
	User       plazaUserRaw `json:"user"`
	CreatedAt  string       `json:"createdAt"`
}

// parseTime 解析ISO时间字符串
func parsePlazaTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// apiGet 调用外部广场GET接口
func (s *PlazaSyncService) apiGet(path string, params url.Values) (json.RawMessage, error) {
	u, _ := url.Parse(s.apiBase + path)
	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResp plazaAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w (body: %s)", err, string(body[:min(len(body), 200)]))
	}
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API错误: %s", apiResp.Message)
	}
	return apiResp.Data, nil
}

// apiPost 调用外部广场POST接口
func (s *PlazaSyncService) apiPost(path string, body interface{}) (json.RawMessage, error) {
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", s.apiBase+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResp plazaAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API错误: %s", apiResp.Message)
	}
	return apiResp.Data, nil
}

// Login 登录外部广场获取Token
func (s *PlazaSyncService) Login(username, password string) error {
	data, err := s.apiPost("/auth/login", map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}

	var loginData plazaLoginData
	if err := json.Unmarshal(data, &loginData); err != nil {
		return fmt.Errorf("解析登录数据失败: %w", err)
	}

	s.token = loginData.Token
	logger.SugaredLogger.Infof("提示词广场登录成功: %s (VIP%d)", loginData.User.Nickname, loginData.User.VipLevel)
	return nil
}

// SyncPrompts 同步提示词数据到本地表
func (s *PlazaSyncService) SyncPrompts() (int, error) {
	if s.token == "" {
		return 0, fmt.Errorf("未登录，请先调用 Login()")
	}

	now := time.Now()
	var allPrompts []models.PlazaPrompt
	page := 1
	pageSize := 50

	for {
		params := url.Values{
			"page":     {strconv.Itoa(page)},
			"pageSize": {strconv.Itoa(pageSize)},
			"sort":     {"latest"},
		}

		data, err := s.apiGet("/prompts", params)
		if err != nil {
			return 0, fmt.Errorf("获取提示词列表第%d页失败: %w", page, err)
		}

		var listData plazaPromptListData
		if err := json.Unmarshal(data, &listData); err != nil {
			return 0, fmt.Errorf("解析提示词列表失败: %w", err)
		}

		if len(listData.List) == 0 {
			break
		}

		for _, raw := range listData.List {
			prompt := models.PlazaPrompt{
				ExtID:          raw.ID,
				Title:          raw.Title,
				Content:        raw.Content,
				Description:    raw.Description,
				Summary:        raw.Summary,
				Category:       raw.Category,
				Tags:           raw.Tags,
				IsPublic:       raw.IsPublic,
				VipOnly:        raw.VipOnly,
				NeedVip:        raw.NeedVip,
				AuthorID:       raw.UserID,
				AuthorNickname: raw.User.Nickname,
				AuthorUsername: raw.User.Username,
				AuthorVipLevel: raw.User.VipLevel,
				ViewsCount:     raw.ViewsCount,
				LikesCount:     raw.LikesCount,
				FavoritesCount: raw.FavoritesCount,
				DownloadsCount: raw.DownloadsCount,
				CommentsCount:  raw.CommentsCount,
				HotScore:       raw.HotScore,
				ExtCreatedAt:   parsePlazaTime(raw.CreatedAt),
				ExtUpdatedAt:   parsePlazaTime(raw.UpdatedAt),
				SyncedAt:       now,
			}
			allPrompts = append(allPrompts, prompt)
		}

		// 检查是否还有下一页
		totalPages := int(listData.Total) / pageSize
		if int(listData.Total)%pageSize > 0 {
			totalPages++
		}
		if page >= totalPages {
			break
		}
		page++
	}

	if len(allPrompts) == 0 {
		logger.SugaredLogger.Info("提示词广场无数据")
		return 0, nil
	}

	// 清空旧数据，全量替换
	if err := db.Dao.Where("1=1").Delete(&models.PlazaPrompt{}).Error; err != nil {
		return 0, fmt.Errorf("清空旧提示词数据失败: %w", err)
	}

	// 批量插入
	if err := db.Dao.CreateInBatches(allPrompts, 100).Error; err != nil {
		return 0, fmt.Errorf("插入提示词数据失败: %w", err)
	}

	logger.SugaredLogger.Infof("同步提示词广场数据完成，共 %d 条", len(allPrompts))
	return len(allPrompts), nil
}

// SyncQuestions 同步问答数据到本地表
func (s *PlazaSyncService) SyncQuestions() (int, error) {
	if s.token == "" {
		return 0, fmt.Errorf("未登录，请先调用 Login()")
	}

	now := time.Now()
	var allQuestions []models.PlazaQuestion
	page := 1
	pageSize := 50

	for {
		params := url.Values{
			"page":     {strconv.Itoa(page)},
			"pageSize": {strconv.Itoa(pageSize)},
		}

		data, err := s.apiGet("/questions", params)
		if err != nil {
			return 0, fmt.Errorf("获取问答列表第%d页失败: %w", page, err)
		}

		var listData plazaQuestionListData
		if err := json.Unmarshal(data, &listData); err != nil {
			return 0, fmt.Errorf("解析问答列表失败: %w", err)
		}

		if len(listData.List) == 0 {
			break
		}

		for _, raw := range listData.List {
			question := models.PlazaQuestion{
				ExtID:          raw.ID,
				Title:          raw.Title,
				Content:        raw.Content,
				IsResolved:     raw.IsResolved,
				AuthorID:       raw.UserID,
				AuthorNickname: raw.User.Nickname,
				AuthorUsername: raw.User.Username,
				AnswersCount:   raw.AnswersCount,
				ExtCreatedAt:   parsePlazaTime(raw.CreatedAt),
				ExtUpdatedAt:   parsePlazaTime(raw.UpdatedAt),
				SyncedAt:       now,
			}

			// 尝试获取问答详情（含回答列表）
			detailData, err := s.apiGet(fmt.Sprintf("/questions/%d", raw.ID), nil)
			if err == nil {
				var detail plazaQuestionDetailData
				if json.Unmarshal(detailData, &detail) == nil && len(detail.Answers) > 0 {
					answersJSON, _ := json.Marshal(detail.Answers)
					question.AnswersJSON = string(answersJSON)
				}
			}

			allQuestions = append(allQuestions, question)
		}

		// 检查是否还有下一页
		totalPages := int(listData.Total) / pageSize
		if int(listData.Total)%pageSize > 0 {
			totalPages++
		}
		if page >= totalPages {
			break
		}
		page++
	}

	if len(allQuestions) == 0 {
		logger.SugaredLogger.Info("问答广场无数据")
		return 0, nil
	}

	// 清空旧数据，全量替换
	if err := db.Dao.Where("1=1").Delete(&models.PlazaQuestion{}).Error; err != nil {
		return 0, fmt.Errorf("清空旧问答数据失败: %w", err)
	}

	// 批量插入
	if err := db.Dao.CreateInBatches(allQuestions, 100).Error; err != nil {
		return 0, fmt.Errorf("插入问答数据失败: %w", err)
	}

	logger.SugaredLogger.Infof("同步问答广场数据完成，共 %d 条", len(allQuestions))
	return len(allQuestions), nil
}

// GetPlazaCategories 从本地表获取分类列表
func GetPlazaCategories() []string {
	var categories []string
	db.Dao.Model(&models.PlazaPrompt{}).Distinct("category").Where("category != ''").Pluck("category", &categories)
	return categories
}

// GetPlazaPrompts 从本地表查询提示词列表
func GetPlazaPrompts(query *models.PlazaPromptQuery) (list []models.PlazaPrompt, total int64, err error) {
	dbQuery := db.Dao.Model(&models.PlazaPrompt{})

	if query.Category != "" {
		dbQuery = dbQuery.Where("category = ?", query.Category)
	}
	if query.Keyword != "" {
		dbQuery = dbQuery.Where("title LIKE ? OR description LIKE ? OR tags LIKE ?",
			"%"+query.Keyword+"%", "%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}
	if query.VipOnly == "true" {
		dbQuery = dbQuery.Where("vip_only = ?", true)
	}

	dbQuery.Count(&total)

	page := query.Page
	pageSize := query.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 12
	}

	// 排序
	switch query.Sort {
	case "hot":
		dbQuery = dbQuery.Order("hot_score DESC")
	case "likes":
		dbQuery = dbQuery.Order("likes_count DESC")
	case "favorites":
		dbQuery = dbQuery.Order("favorites_count DESC")
	case "downloads":
		dbQuery = dbQuery.Order("downloads_count DESC")
	case "comments":
		dbQuery = dbQuery.Order("comments_count DESC")
	default: // latest
		dbQuery = dbQuery.Order("ext_created_at DESC")
	}

	err = dbQuery.Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return
}

// GetPlazaPromptByExtID 根据外部ID获取提示词详情
func GetPlazaPromptByExtID(extID uint) (*models.PlazaPrompt, error) {
	var prompt models.PlazaPrompt
	err := db.Dao.Where("ext_id = ?", extID).First(&prompt).Error
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

// GetPlazaQuestions 从本地表查询问答列表
func GetPlazaQuestions(query *models.PlazaQuestionQuery) (list []models.PlazaQuestion, total int64, err error) {
	dbQuery := db.Dao.Model(&models.PlazaQuestion{})

	if query.Keyword != "" {
		dbQuery = dbQuery.Where("title LIKE ? OR content LIKE ?",
			"%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}
	if query.Resolved == "true" {
		dbQuery = dbQuery.Where("is_resolved = ?", true)
	} else if query.Resolved == "false" {
		dbQuery = dbQuery.Where("is_resolved = ?", false)
	}

	dbQuery.Count(&total)

	page := query.Page
	pageSize := query.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 15
	}

	err = dbQuery.Offset((page - 1) * pageSize).Limit(pageSize).Order("ext_created_at DESC").Find(&list).Error
	return
}

// GetPlazaQuestionByExtID 根据外部ID获取问答详情
func GetPlazaQuestionByExtID(extID uint) (*models.PlazaQuestion, error) {
	var question models.PlazaQuestion
	err := db.Dao.Where("ext_id = ?", extID).First(&question).Error
	if err != nil {
		return nil, err
	}
	return &question, nil
}

// GetLastPlazaSyncTime 获取最近一次同步时间
func GetLastPlazaSyncTime() time.Time {
	var prompt models.PlazaPrompt
	if err := db.Dao.Order("synced_at DESC").First(&prompt).Error; err != nil {
		return time.Time{}
	}
	return prompt.SyncedAt
}
