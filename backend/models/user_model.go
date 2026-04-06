package models

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

// User 用户模型
type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt soft_delete.DeletedAt `gorm:"index" json:"-"`

	Username    string `gorm:"uniqueIndex;not null" json:"username"`
	Password    string `gorm:"not null" json:"password"` // 应该存储哈希值
	Email       string `gorm:"uniqueIndex" json:"email"`
	DisplayName string `json:"displayName"`
	IsActive    bool   `gorm:"default:true" json:"isActive"`

	// 用户偏好设置
	Settings map[string]interface{} `gorm:"serializer:json" json:"settings"`
}

func (User) TableName() string {
	return "users"
}