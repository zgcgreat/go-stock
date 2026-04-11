// cmd/web/main.go - go-stock Web 服务器独立启动入口
//
// 编译方法（在项目根目录执行）：
//   go build -o go-stock-web.exe ./cmd/web
//
// 运行方法（前端 dist 需要在当前目录或 GO_STOCK_STATIC_DIR 指定目录）：
//   GO_STOCK_WEB_PORT=8080 ./go-stock-web.exe

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/internal/handlers"
	"go-stock/internal/webserver"
)

func main() {
	port := os.Getenv("GO_STOCK_WEB_PORT")
	if port == "" {
		port = "8080"
	}

	// 数据库初始化
	dbPath := os.Getenv("GO_STOCK_DB_PATH")
	db.Init(dbPath)

	// 初始化情感分析
	data.InitAnalyzeSentiment()

	// 数据库迁移
	webserver.MigrateAllTables()

	// 确保初始管理员账户存在
	handlers.AdminEnsureInitialAdmin()

	// 设置静态资源目录（磁盘路径，默认 frontend/dist）
	if staticDir := os.Getenv("GO_STOCK_STATIC_DIR"); staticDir != "" {
		webserver.StaticDir = staticDir
	}
	// WebAssets 为 nil，将使用 StaticDir 磁盘路径

	// 启动 Web 服务器
	server := webserver.NewWebServer(port)
	log.Printf("go-stock Web server starting on http://0.0.0.0:%s", port)

	// 启动定时任务调度器（市场统计、热词采集等）
	scheduler := webserver.GetScheduler()
	scheduler.Start()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		scheduler.Stop()
		os.Exit(0)
	}()

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
