package main

import (
	"fmt"
	"log"

	"go-stock/backend/db"
	"go-stock/backend/models"
)

func main() {
	// 初始化数据库
	db.Init("")

	// 查找admin用户
	var adminUser models.User
	if err := db.Dao.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		log.Fatalf("未找到admin用户: %v", err)
	}

	fmt.Printf("找到用户: %s, 当前角色: %s\n", adminUser.Username, adminUser.Role)

	// 更新角色为admin
	if err := db.Dao.Model(&adminUser).Update("role", "super_admin").Error; err != nil {
		log.Fatalf("更新失败: %v", err)
	}

	fmt.Println("✅ 成功将admin用户的角色更新为 'admin'")

	// 验证更新
	var updatedUser models.User
	if err := db.Dao.Where("username = ?", "admin").First(&updatedUser).Error; err == nil {
		fmt.Printf("验证: 用户 %s 的角色现在是: %s\n", updatedUser.Username, updatedUser.Role)
	}
}
