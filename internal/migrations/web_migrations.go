package migrations

import (
	"fmt"
	"log"

	"go-stock/backend/db"
)

// ApplyWebCompatibilityMigrations applies SQLite column patches required by the Web edition.
// Keep these Web-specific compatibility patches outside webserver route registration to reduce
// conflicts when syncing upstream server.go changes.
func ApplyWebCompatibilityMigrations() {
	migrateUserIDColumns([]string{
		"settings",              // data.Settings
		"ai_config",             // data.AIConfig
		"followed_stock",        // data.FollowedStock
		"trading_records",       // data.TradingRecord
		"cron_tasks",            // models.CronTask
		"ai_assistant_sessions", // models.AiAssistantSession
		"ai_recommend_stocks",   // models.AiRecommendStocks
		"followed_fund",         // data.FollowedFund
		"stock_groups",          // data.Group
		"group_stock_info",      // data.GroupStock
		"ai_response_result",    // models.AIResponseResult
		"daily_operation_plan",  // models.DailyOperationPlan
	})

	migrateSettingsMissingColumns()
	migrateAIConfigMissingColumns()
}

func migrateUserIDColumns(tableNames []string) {
	for _, tableName := range tableNames {
		migrateUserIDColumn(tableName)
	}
}

// migrateUserIDColumn ensures that a table has a user_id column.
func migrateUserIDColumn(tableName string) {
	var count int64
	db.Dao.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", tableName, "user_id").Scan(&count)
	if count == 0 {
		db.Dao.Exec("ALTER TABLE " + tableName + " ADD COLUMN user_id INTEGER DEFAULT 0")
		log.Printf("Migrated: Added user_id column to %s", tableName)
	}
}

// migrateSettingsMissingColumns ensures that settings includes fields added after initial Web deployment.
// GORM AutoMigrate may not add columns reliably on SQLite in older deployed databases.
func migrateSettingsMissingColumns() {
	tableName := "settings"
	columns := map[string]string{
		"update_channel":            "VARCHAR(20) DEFAULT ''",
		"prompt_plaza_api_base":     "VARCHAR(255) DEFAULT ''",
		"browser_pool_size":         "INTEGER DEFAULT 0",
		"enable_push_news":          "INTEGER DEFAULT 0",
		"enable_only_push_red_news": "INTEGER DEFAULT 0",
		"qgqp_b_id":                 "VARCHAR(100) DEFAULT ''",
		"iwencai_api_key":           "VARCHAR(255) DEFAULT ''",
		"em_api_key":                "VARCHAR(255) DEFAULT ''",
		"window_width":              "INTEGER DEFAULT 0",
		"window_height":             "INTEGER DEFAULT 0",
		"http_proxy":                "TEXT DEFAULT ''",
		"http_proxy_enabled":        "INTEGER DEFAULT 0",
		"enable_agent":              "INTEGER DEFAULT 0",
	}

	migrateMissingColumns(tableName, columns)
}

// migrateAIConfigMissingColumns ensures ai_config includes fields added after early deployments.
func migrateAIConfigMissingColumns() {
	tableName := "ai_config"
	columns := map[string]string{
		"user_id":            "INTEGER DEFAULT 0",
		"http_proxy":         "TEXT DEFAULT ''",
		"http_proxy_enabled": "INTEGER DEFAULT 0",
		"session_id":         "VARCHAR(64) DEFAULT ''",
		"thinking":           "INTEGER DEFAULT 0",
	}

	migrateMissingColumns(tableName, columns)
}

func migrateMissingColumns(tableName string, columns map[string]string) {
	for colName, colType := range columns {
		var count int64
		db.Dao.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", tableName, colName).Scan(&count)
		if count == 0 {
			sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, colName, colType)
			db.Dao.Exec(sql)
			log.Printf("Migrated: Added column %s to %s", colName, tableName)
		}
	}
}
