package middleware

import (
	"errors"
	"log"
	"go-stock/backend/db"
	"go-stock/backend/models"
)

// Role 角色类型
type Role string

const (
	RoleAdmin     Role = "admin"      // 管理员
	RoleUser      Role = "user"       // 普通用户
	RoleVIP       Role = "vip"        // VIP用户
	RoleSuperAdmin Role = "super_admin" // 超级管理员
)

// Permission 权限定义
type Permission string

const (
	PermUserRead    Permission = "user:read"
	PermUserWrite   Permission = "user:write"
	PermUserDelete  Permission = "user:delete"
	PermAdminAccess Permission = "admin:access"
)

// rolePermissions 角色与权限的映射
var rolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		PermUserRead, PermUserWrite, PermUserDelete, PermAdminAccess,
	},
	RoleAdmin: {
		PermUserRead, PermUserWrite, PermAdminAccess,
	},
	RoleVIP: {
		PermUserRead,
	},
	RoleUser: {
		PermUserRead,
	},
}

// HasPermission 检查用户是否有指定权限
func HasPermission(userID uint, permission Permission) (bool, error) {
	var user models.User
	if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
		return false, err
	}

	// 获取用户角色
	role := GetUserRole(user)

	// 检查角色是否有该权限
	perms, exists := rolePermissions[role]
	if !exists {
		return false, errors.New("未知角色")
	}

	for _, p := range perms {
		if p == permission {
			return true, nil
		}
	}

	return false, nil
}

// IsAdmin 检查用户是否为管理员
func IsAdmin(userID uint) (bool, error) {
	var user models.User
	if err := db.Dao.Where("id = ?", userID).First(&user).Error; err != nil {
		log.Printf("[IsAdmin] 查询用户失败: userID=%d, err=%v", userID, err)
		return false, err
	}
	log.Printf("[IsAdmin] 用户信息: userID=%d, username=%s, role=%s", userID, user.Username, user.Role)
	role := GetUserRole(user)
	log.Printf("[IsAdmin] 解析角色: %s, 是否管理员: %v", role, role == RoleAdmin || role == RoleSuperAdmin)
	return role == RoleAdmin || role == RoleSuperAdmin, nil
}

// GetUserRole 根据用户信息获取角色
func GetUserRole(user models.User) Role {
	switch user.Role {
	case "super_admin":
		return RoleSuperAdmin
	case "admin":
		return RoleAdmin
	case "vip":
		return RoleVIP
	default:
		return RoleUser
	}
}

// SetUserRole 设置用户角色
func SetUserRole(userID uint, role Role) error {
	return db.Dao.Model(&models.User{}).Where("id = ?", userID).Update("role", string(role)).Error
}
