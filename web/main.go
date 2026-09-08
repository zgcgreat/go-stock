// cmd/web/main.go - go-stock Web 鏈嶅姟鍣ㄧ嫭绔嬪惎鍔ㄥ叆鍙?//
// 缂栬瘧鏂规硶锛堝湪椤圭洰鏍圭洰褰曟墽琛岋級锛?//   go build -o go-stock-web.exe ./cmd/web
//
// 杩愯鏂规硶锛堝墠绔?dist 闇€瑕佸湪褰撳墠鐩綍鎴?GO_STOCK_STATIC_DIR 鎸囧畾鐩綍锛夛細
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

	// 鏁版嵁搴撳垵濮嬪寲
	dbPath := os.Getenv("GO_STOCK_DB_PATH")
	db.Init(dbPath)

	// 鍒濆鍖栨儏鎰熷垎鏋?	data.InitAnalyzeSentiment()

	// 鏁版嵁搴撹縼绉?	webserver.MigrateAllTables()

	// 纭繚鍒濆绠＄悊鍛樿处鎴峰瓨鍦?	handlers.AdminEnsureInitialAdmin()

	// 璁剧疆闈欐€佽祫婧愮洰褰曪紙纾佺洏璺緞锛岄粯璁?frontend/dist锛?	if staticDir := os.Getenv("GO_STOCK_STATIC_DIR"); staticDir != "" {
		webserver.StaticDir = staticDir
	}
	// WebAssets 涓?nil锛屽皢浣跨敤 StaticDir 纾佺洏璺緞

	// 鍚姩 Web 鏈嶅姟鍣?	server := webserver.NewWebServer(port)
	log.Printf("go-stock Web server starting on http://0.0.0.0:%s", port)

	// 鍚姩瀹氭椂浠诲姟璋冨害鍣紙甯傚満缁熻銆佺儹璇嶉噰闆嗙瓑锛?	scheduler := webserver.GetScheduler()
	scheduler.Start()

	// 浼橀泤鍏抽棴
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
