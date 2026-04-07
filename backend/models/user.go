package models

import (
	"gorm.io/plugin/soft_delete"
	"time"
)

// User 用户模型
type User struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Username    string `json:"username" gorm:"uniqueIndex;size:20"`
	Email       string `json:"email" gorm:"uniqueIndex;size:50"`
	Password    string `json:"password" gorm:"not null"`
	DisplayName string `json:"display_name" gorm:"size:50"`
	Role        string `json:"role" gorm:"size:20;default:'user'"` // user, vip, admin, super_admin
	IsActive    bool   `json:"is_active" gorm:"default:true"`
	// VIP相关字段
	VipLevel   int                   `json:"vip_level" gorm:"default:0"` // VIP等级: 0-普通用户, 1-4-VIP等级
	VipStartAt *time.Time            `json:"vip_start_at" gorm:"index"`  // VIP开始时间
	VipEndAt   *time.Time            `json:"vip_end_at" gorm:"index"`    // VIP到期时间
	CreatedAt  time.Time             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time             `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt  soft_delete.DeletedAt `json:"deleted_at" gorm:"index"`
}
