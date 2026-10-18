package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
)

// SSE 事件总线（004 批3）：桌面模式「Go cron 60s → Wails 事件」节拍的 Web 等价物。
//
// 浏览器隐藏标签页会深度节流前端定时器（信号引擎兜底定时器由此退化），但网络事件回调
// 不受该节流影响——因此由服务端 hub 每 60s 向所有在线连接广播 signalMonitorTick，
// 前端桥接层解析后按同款 CustomEvent 分发，下游 EventsOn 消费者（signalMonitor.ts）零改动。
//
// 设计约束：
//   - 服务端是 dumb pipe：不判交易时段、不看用户配置（与桌面一致，门控全在引擎侧）；
//   - 广播对所有认证连接一视同仁（tick 本身无载荷，各标签页自行扫描自己的监控池）；
//   - promptBacktestProgress 等其他桌面事件后续可经 broadcastEvent 并入（004 §6 预留）。

var (
	eventBusMu      sync.Mutex
	eventBusClients = make(map[chan string]struct{})

	// signalTickMu 保护信号节拍循环的热更新（007：扫描间隔 30~600s 可配置）
	signalTickMu   sync.Mutex
	signalTickStop chan struct{}
)

// EventsStreamHandler 服务端事件流：GET /events/stream（JWT 保护，fetch-stream 读取，
// 不用 EventSource——后者无法携带 Authorization 头）。
func EventsStreamHandler(c *gin.Context) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "streaming unsupported"})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	ch := make(chan string, 8)
	eventBusMu.Lock()
	eventBusClients[ch] = struct{}{}
	eventBusMu.Unlock()
	defer func() {
		eventBusMu.Lock()
		delete(eventBusClients, ch)
		eventBusMu.Unlock()
	}()
	startSignalTickLoop()

	// 立即冲刷响应头并写一行注释作为握手确认（SSE 注释行对客户端不可见）
	_, _ = fmt.Fprint(c.Writer, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg := <-ch:
			if _, err := fmt.Fprint(c.Writer, msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// startSignalTickLoop 节拍广播器（进程级单例；空连接时 ticker 空转，零成本）。
// 节拍间隔取设置页「扫描间隔」（默认 60s），经 SetSignalScanInterval API 热更新。
func startSignalTickLoop() {
	signalTickMu.Lock()
	defer signalTickMu.Unlock()
	if signalTickStop != nil {
		return // 已有循环在跑
	}
	signalTickStop = make(chan struct{})
	go signalTickLoop(data.GetSignalScanIntervalSec(), signalTickStop)
}

// signalTickLoop 按给定间隔广播 signalMonitorTick；stop 被关闭后退出（由热更新重启新循环）。
func signalTickLoop(sec int, stop chan struct{}) {
	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			BroadcastEvent("event: signalMonitorTick\ndata: {}\n\n")
		}
	}
}

// RestartSignalTickInterval 热更新信号节拍间隔：停止旧循环并以新间隔重启（幂等）。
func RestartSignalTickInterval(sec int) {
	signalTickMu.Lock()
	defer signalTickMu.Unlock()
	if signalTickStop != nil {
		close(signalTickStop)
	}
	signalTickStop = make(chan struct{})
	go signalTickLoop(sec, signalTickStop)
}

// BroadcastEvent 向所有在线连接广播一条 SSE 消息；慢消费者（缓冲满）丢弃本条——
// tick 是幂等节拍，丢一条等下一轮即可，不阻塞其他连接。
func BroadcastEvent(msg string) {
	eventBusMu.Lock()
	defer eventBusMu.Unlock()
	for ch := range eventBusClients {
		select {
		case ch <- msg:
		default:
		}
	}
}
