package db

import (
	"gorm.io/gorm"
)

// DatabaseInterface 数据库接口定义，便于切换不同数据库驱动
type DatabaseInterface interface {
	Init(dbHost, dbName, dbUser, dbPass string, maxOpenConns, maxIdleConns int) error
	GetDB() *gorm.DB
	Close() error
}

// MultiTenantDB 扩展数据库功能，支持多租户
type MultiTenantDB struct {
	DB *gorm.DB
}

// WithUserFilter 为查询添加用户过滤条件
func (mt *MultiTenantDB) WithUserFilter(userID uint) *gorm.DB {
	return mt.DB.Where("user_id = ?", userID)
}

// GetAllUserRecords 获取特定用户的全部记录
func (mt *MultiTenantDB) GetAllUserRecords(model interface{}, userID uint) *gorm.DB {
	return mt.DB.Where("user_id = ?", userID).Find(model)
}

// GetUserRecordByID 根据ID和用户获取特定记录
func (mt *MultiTenantDB) GetUserRecordByID(model interface{}, id uint, userID uint) *gorm.DB {
	return mt.DB.Where("id = ? AND user_id = ?", id, userID).First(model)
}

// CreateUserRecord 创建用户的记录
func (mt *MultiTenantDB) CreateUserRecord(model interface{}) *gorm.DB {
	return mt.DB.Create(model)
}