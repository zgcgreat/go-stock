package models

import (
	"time"
	"gorm.io/plugin/soft_delete"
)

// User 用户模型
type User struct {
	ID          uint              `json:"id" gorm:"primaryKey"`
	Username    string            `json:"username" gorm:"uniqueIndex;size:20"`
	Email       string            `json:"email" gorm:"uniqueIndex;size:50"`
	Password    string            `json:"password" gorm:"not null"`
	DisplayName string            `json:"display_name" gorm:"size:50"`
	IsActive    bool              `json:"is_active" gorm:"default:true"`
	IsAdmin     bool              `json:"is_admin" gorm:"default:false"`
	CreatedAt   time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   soft_delete.DeletedAt `json:"deleted_at" gorm:"index"`
}