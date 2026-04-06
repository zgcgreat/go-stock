package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"go-stock/backend/db"
	"go-stock/backend/models"
	"go-stock/internal/middleware"
)

// AdminUserManager 管理管理员相关功能
type AdminUserManager struct{}

// CreateInitialAdmin 创建初始管理员账户
func (m *AdminUserManager) CreateInitialAdmin(username, email, password, displayName string) error {
	// 检查是否已有管理员
	var userCount int64
	db.Dao.Model(&models.User{}).Count(&userCount)

	if userCount > 0 {
		return nil // 如果已有用户则不创建默认管理员
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	newUser := &models.User{
		Username:    username,
		Email:       email,
		Password:    string(hashedPassword),
		DisplayName: displayName,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return db.Dao.Create(newUser).Error
}

// UpdateUserStatus 更新用户状态（激活/停用）
func (m *AdminUserManager) UpdateUserStatus(userID uint, isActive bool) error {
	var user models.User
	if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	return db.Dao.Model(&user).Update("is_active", isActive).Error
}

// GetUserList 获取用户列表
func (m *AdminUserManager) GetUserList(page, pageSize int, searchKeyword string) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	query := db.Dao.Model(&models.User{})

	if searchKeyword != "" {
		keyword := "%" + searchKeyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR display_name LIKE ?", keyword, keyword, keyword)
	}

	query.Count(&total)

	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// DeleteUser 删除用户
func (m *AdminUserManager) DeleteUser(userID uint) error {
	return db.Dao.Delete(&models.User{}, "id = ?", userID).Error
}

// ResetUserPassword 重置用户密码
func (m *AdminUserManager) ResetUserPassword(userID uint, newPassword string) error {
	var user models.User
	if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return db.Dao.Model(&user).Update("password", string(hashedPassword)).Error
}

// AdminEnsureInitialAdmin 确保存在初始管理员账户
func AdminEnsureInitialAdmin() {
	manager := &AdminUserManager{}

	// 检查是否有用户存在
	var userCount int64
	db.Dao.Model(&models.User{}).Count(&userCount)
	println("现有用户总数:", userCount)

	// 如果没有用户，则创建一个默认管理员
	if userCount == 0 {
		adminPassword := "admin123" // 初始默认密码，在生产环境中应该更改
		err := manager.CreateInitialAdmin("admin", "admin@example.com", adminPassword, "系统管理员")
		if err != nil {
			// 记录错误日志，但不中断程序执行
			println("创建初始管理员账户失败:", err.Error())
		} else {
			// 设置 IsAdmin 为 true
			var adminUser models.User
			if err := db.Dao.Where("username = ?", "admin").First(&adminUser).Error; err == nil {
				db.Dao.Model(&adminUser).Update("is_admin", true)
			}
			println("初始管理员账户创建成功！用户名: admin, 密码: admin123")
		}
	} else {
		// 即使有用户存在，也可以检查是否存在管理员用户
		var adminUserCount int64
		db.Dao.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminUserCount)
		println("现有管理员数量:", adminUserCount)

		if adminUserCount == 0 {
			println("警告：没有发现管理员用户，可能需要手动创建或修复管理员角色")
		} else {
			var adminUsers []models.User
			db.Dao.Model(&models.User{}).Where("is_admin = ?", true).Find(&adminUsers)
			for _, user := range adminUsers {
				println("发现管理员用户:", user.Username, "ID:", user.ID)
			}
		}
	}
}

// AdminUserList 获取用户列表
func AdminUserList(c *gin.Context) {
	// 权限检查已由中间件 AdminRequired 完成
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}

	searchKeyword := c.Query("search")

	// 添加调试日志
	println("AdminUserList called with page:", page, "pageSize:", pageSize, "search:", searchKeyword)

	manager := &AdminUserManager{}
	users, total, err := manager.GetUserList(page, pageSize, searchKeyword)
	if err != nil {
		println("Error getting user list:", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取用户列表失败",
			"message": "查询用户列表出现错误",
		})
		return
	}

	println("Successfully retrieved users:", len(users), "total:", total)

	totalPages := total / int64(pageSize)
	if total%int64(pageSize) > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":       users,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// UpdateUserActiveStatus 更新用户激活状态
func UpdateUserActiveStatus(c *gin.Context) {
	// 权限检查已由中间件 AdminRequired 完成
	var req struct {
		UserID uint `json:"userId" binding:"required"`
		Active bool `json:"active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求数据错误",
			"message": err.Error(),
		})
		return
	}

	manager := &AdminUserManager{}
	err := manager.UpdateUserStatus(req.UserID, req.Active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "更新用户状态失败",
			"message": "更新用户状态出现错误",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户状态更新成功",
	})
}

// CreateUser 创建新用户
func CreateUser(c *gin.Context) {
	// 权限检查已由中间件 AdminRequired 完成
	var req struct {
		Username    string `json:"username" binding:"required,min=3,max=20"`
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=6"`
		DisplayName string `json:"display_name,omitempty"`
		IsActive    bool   `json:"is_active"`
		IsAdmin     bool   `json:"is_admin"` // 允许设置管理员权限
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求数据错误",
			"message": err.Error(),
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "密码加密失败",
			"message": "内部服务器错误",
		})
		return
	}

	newUser := &models.User{
		Username:    req.Username,
		Email:       req.Email,
		Password:    string(hashedPassword),
		DisplayName: req.DisplayName,
		IsActive:    req.IsActive,
		IsAdmin:     req.IsAdmin,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Dao.Create(newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "创建用户失败",
			"message": "用户创建出现错误",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "用户创建成功",
		"data": gin.H{
			"id":            newUser.ID,
			"username":      newUser.Username,
			"display_name":  newUser.DisplayName,
			"email":         newUser.Email,
			"is_active":     newUser.IsActive,
			"is_admin":      newUser.IsAdmin,
			"created_at":    newUser.CreatedAt,
			"updated_at":    newUser.UpdatedAt,
		},
	})
}

// DeleteUserHandler 删除用户
func DeleteUserHandler(c *gin.Context) {
	// 权限检查已由中间件 AdminRequired 完成
	currentUserID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "未找到用户信息",
			"message": "认证失败",
		})
		return
	}

	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"message": "无效的用户ID",
		})
		return
	}

	// 不能删除自己
	if uint(userID) == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "操作不允许",
			"message": "不能删除当前登录的管理员账户",
		})
		return
	}

	manager := &AdminUserManager{}
	err = manager.DeleteUser(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "删除用户失败",
			"message": "删除用户出现错误",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "用户删除成功",
	})
}

// ResetUserPasswordHandler 重置用户密码
func ResetUserPasswordHandler(c *gin.Context) {
	// 权限检查已由中间件 AdminRequired 完成
	var req struct {
		UserID      uint   `json:"userId" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求数据错误",
			"message": err.Error(),
		})
		return
	}

	manager := &AdminUserManager{}
	err := manager.ResetUserPassword(req.UserID, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "重置密码失败",
			"message": "重置密码出现错误",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码重置成功",
	})
}