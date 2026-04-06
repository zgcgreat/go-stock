package models

import (
	"time"
	"gorm.io/plugin/soft_delete"
)

// Base 基础模型，用于多租户支持
type Base struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"userId"` // 添加多租户支持的用户标识
	IsDel     soft_delete.DeletedAt `gorm:"softDelete:flag" json:"-"`
}

// BaseNoDel 基础模型（不含软删除）
type BaseNoDel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"userId"` // 添加多租户支持的用户标识
}