package data

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"go-stock/backend/models"
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

// EnsureInitialAdmin 确保存在初始管理员账户
func EnsureInitialAdmin() {
	manager := &AdminUserManager{}

	// 检查是否有用户存在
	var userCount int64
	db.Dao.Model(&models.User{}).Count(&userCount)

	// 如果没有用户，则创建一个默认管理员
	if userCount == 0 {
		adminPassword := "admin123" // 初始默认密码，在生产环境中应该更改
		err := manager.CreateInitialAdmin("admin", "admin@example.com", adminPassword, "系统管理员")
		if err != nil {
			// 记录错误日志，但不中断程序执行
			println("创建初始管理员账户失败:", err.Error())
		} else {
			println("初始管理员账户创建成功！用户名: admin, 密码: admin123")
		}
	}
}