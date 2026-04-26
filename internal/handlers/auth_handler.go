package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go-stock/backend/db"
	"go-stock/backend/logger"
	"go-stock/backend/models"
	"go-stock/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token      string `json:"token"`
	UserID     uint   `json:"userId"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	ExpiresAt  int64  `json:"expiresAt"`
	VipLevel   int    `json:"vipLevel"`
	VipStartAt string `json:"vipStartAt,omitempty"`
	VipEndAt   string `json:"vipEndAt,omitempty"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

// HandleLogin 处理用户登录
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "无效的请求体")
		return
	}

	// 查询用户
	var user models.User
	if err := db.Dao.Where("username = ?", req.Username).First(&user).Error; err != nil {
		writeJSONError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误")
		return
	}

	// 检查用户是否激活
	if !user.IsActive {
		writeJSONError(w, http.StatusForbidden, "ACCOUNT_DISABLED", "账户已被禁用")
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误")
		return
	}

	// 生成JWT token
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		logger.SugaredLogger.Errorf("生成token失败: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "TOKEN_ERROR", "生成认证令牌失败")
		return
	}

	// 计算过期时间（24小时）
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	// 格式化VIP时间
	var vipStartAt, vipEndAt string
	if user.VipStartAt != nil && !user.VipStartAt.IsZero() {
		vipStartAt = user.VipStartAt.Format("2006-01-02 15:04:05")
	}
	if user.VipEndAt != nil && !user.VipEndAt.IsZero() {
		vipEndAt = user.VipEndAt.Format("2006-01-02 15:04:05")
	}

	// 返回登录成功信息
	response := LoginResponse{
		Token:      token,
		UserID:     user.ID,
		Username:   user.Username,
		Role:       user.Role,
		ExpiresAt:  expiresAt,
		VipLevel:   user.VipLevel,
		VipStartAt: vipStartAt,
		VipEndAt:   vipEndAt,
	}

	writeJSON(w, http.StatusOK, response)
}

// HandleRegister 处理用户注册
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "无效的请求体")
		return
	}

	// 验证必填字段
	if req.Username == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "MISSING_FIELDS", "用户名和密码不能为空")
		return
	}

	// 检查用户名是否已存在
	var existingUser models.User
	if err := db.Dao.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		writeJSONError(w, http.StatusConflict, "USERNAME_EXISTS", "用户名已存在")
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.SugaredLogger.Errorf("密码加密失败: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "PASSWORD_HASH_ERROR", "密码加密失败")
		return
	}

	// 创建新用户
	user := models.User{
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hashedPassword),
		DisplayName: req.DisplayName,
		Role:        "user", // 默认角色为普通用户
		IsActive:    true,
	}

	if err := db.Dao.Create(&user).Error; err != nil {
		logger.SugaredLogger.Errorf("创建用户失败: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "CREATE_USER_ERROR", "创建用户失败")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "注册成功",
		"userId":  user.ID,
	})
}

// HandleGetUserProfile 获取当前用户信息
func HandleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 从context中获取用户ID
	userID, ok := middleware.GetUserIDFromHTTPContext(r)
	if !ok || userID == 0 {
		writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "需要登录认证")
		return
	}

	// 查询用户信息
	var user models.User
	if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
		writeJSONError(w, http.StatusNotFound, "USER_NOT_FOUND", "用户不存在")
		return
	}

	// 返回用户信息（不包含密码）
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"displayName": user.DisplayName,
		"role":        user.Role,
		"isActive":    user.IsActive,
		"createdAt":   user.CreatedAt,
	})
}

// writeJSON 写入JSON响应
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeJSONError 写入JSON错误响应
func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	response := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	_ = json.NewEncoder(w).Encode(response)
}

// ==================== Gin Handler 包装函数 ====================

// Register 用户注册 (Gin handler)
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体", "message": err.Error()})
		return
	}

	// 验证必填字段
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MISSING_FIELDS", "message": "用户名和密码不能为空"})
		return
	}

	// 检查用户名是否已存在
	var existingUser models.User
	if err := db.Dao.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "USERNAME_EXISTS", "message": "用户名已存在"})
		return
	}

	// 检查邮箱是否已存在
	if req.Email != "" {
		if err := db.Dao.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "EMAIL_EXISTS", "message": "邮箱已被注册"})
			return
		}
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.SugaredLogger.Errorf("密码加密失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "PASSWORD_HASH_ERROR", "message": "密码加密失败"})
		return
	}

	// 创建新用户
	user := models.User{
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hashedPassword),
		DisplayName: req.DisplayName,
		Role:        "user",
		IsActive:    true,
	}

	if err := db.Dao.Create(&user).Error; err != nil {
		logger.SugaredLogger.Errorf("创建用户失败: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE constraint failed: users.email") {
			c.JSON(http.StatusConflict, gin.H{"error": "EMAIL_EXISTS", "message": "邮箱已被注册"})
		} else if strings.Contains(errMsg, "UNIQUE constraint failed: users.username") {
			c.JSON(http.StatusConflict, gin.H{"error": "USERNAME_EXISTS", "message": "用户名已存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "CREATE_USER_ERROR", "message": "创建用户失败"})
		}
		return
	}

	// 注册成功后生成 token
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		logger.SugaredLogger.Errorf("生成token失败: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功",
		"userId":  user.ID,
		"token":  token,
	})
}

// Login 用户登录 (Gin handler)
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "message": "无效的请求体"})
		return
	}

	// 查询用户
	var user models.User
	if err := db.Dao.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "INVALID_CREDENTIALS", "message": "用户名或密码错误"})
		return
	}

	// 检查用户是否激活
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "ACCOUNT_DISABLED", "message": "账户已被禁用"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "INVALID_CREDENTIALS", "message": "用户名或密码错误"})
		return
	}

	// 生成JWT token
	token, err := middleware.GenerateToken(&user)
	if err != nil {
		logger.SugaredLogger.Errorf("生成token失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TOKEN_ERROR", "message": "生成认证令牌失败"})
		return
	}

	// 计算过期时间（24小时）
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	// 格式化VIP时间
	var vipStartAt, vipEndAt string
	if user.VipStartAt != nil && !user.VipStartAt.IsZero() {
		vipStartAt = user.VipStartAt.Format("2006-01-02 15:04:05")
	}
	if user.VipEndAt != nil && !user.VipEndAt.IsZero() {
		vipEndAt = user.VipEndAt.Format("2006-01-02 15:04:05")
	}

	// 返回登录成功信息
	c.JSON(http.StatusOK, LoginResponse{
		Token:      token,
		UserID:     user.ID,
		Username:   user.Username,
		Role:       user.Role,
		ExpiresAt:  expiresAt,
		VipLevel:   user.VipLevel,
		VipStartAt: vipStartAt,
		VipEndAt:   vipEndAt,
	})
}

// Logout 用户登出 (Gin handler)
func Logout(c *gin.Context) {
	// JWT是无状态的，客户端只需删除token即可
	// 这里可以添加token黑名单逻辑（可选）
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// GetUserProfile 获取当前用户信息 (Gin handler)
func GetUserProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "UNAUTHORIZED", "message": "需要登录认证"})
		return
	}

	// 查询用户信息
	var user models.User
	if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "USER_NOT_FOUND", "message": "用户不存在"})
		return
	}

	// 格式化VIP时间
	var vipStartAt, vipEndAt string
	if user.VipStartAt != nil && !user.VipStartAt.IsZero() {
		vipStartAt = user.VipStartAt.Format("2006-01-02 15:04:05")
	}
	if user.VipEndAt != nil && !user.VipEndAt.IsZero() {
		vipEndAt = user.VipEndAt.Format("2006-01-02 15:04:05")
	}

	// 返回用户信息（包含VIP字段）
	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"displayName": user.DisplayName,
		"role":        user.Role,
		"isActive":    user.IsActive,
		"createdAt":   user.CreatedAt,
		"vipLevel":    user.VipLevel,
		"vipStartAt":  vipStartAt,
		"vipEndAt":    vipEndAt,
	})
}

// UpdateUserProfile 更新用户信息 (Gin handler)
func UpdateUserProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "UNAUTHORIZED", "message": "需要登录认证"})
		return
	}

	var updateData struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "message": "无效的请求体"})
		return
	}

	// 更新用户信息
	updates := make(map[string]interface{})
	if updateData.Email != "" {
		updates["email"] = updateData.Email
	}
	if updateData.DisplayName != "" {
		updates["display_name"] = updateData.DisplayName
	}

	if len(updates) > 0 {
		if err := db.Dao.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			logger.SugaredLogger.Errorf("更新用户信息失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "UPDATE_FAILED", "message": "更新失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}
