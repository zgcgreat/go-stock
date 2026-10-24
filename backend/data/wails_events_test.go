package data

import (
	"context"
	"testing"
)

// TestHasWailsEvents 锁定 Web 端崩溃的判定依据：只有携带 wails "events" 键的
// 生命周期 ctx 才允许进入 runtime.EventsEmit。
//
// 回归背景：Web 端手动生成复盘报告时 handler 传入 context.Background()，
// wails getEvents(ctx) 找不到 events 键即 log.Fatalf → os.Exit(1)，recover 无效，
// 导致整个服务进程被杀（exit status 1）。
func TestHasWailsEvents(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "nil ctx", ctx: nil, want: false},
		{name: "context.Background", ctx: context.Background(), want: false},
		{name: "context.TODO", ctx: context.TODO(), want: false},
		{name: "canceled 无 events 键", ctx: func() context.Context {
			c, cancel := context.WithCancel(context.Background())
			cancel()
			return c
		}(), want: false},
		{name: "带 events 键的 ctx", ctx: context.WithValue(context.Background(), wailsEventsCtxKey, struct{}{}), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasWailsEvents(tt.ctx); got != tt.want {
				t.Fatalf("HasWailsEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSafeEventsEmitNoFatal 验证无 events 键时 SafeEventsEmit 直接返回、不触发
// wails 的 log.Fatalf。这是本缺陷的核心断言：调用 SafeEventsEmit 后进程必须存活。
func TestSafeEventsEmitNoFatal(t *testing.T) {
	// 若 SafeEventsEmit 未做前置校验，此处会 os.Exit(1) 导致测试进程被杀，
	// 表现为 go test 整体失败且无正常的断言输出。
	SafeEventsEmit(context.Background(), "dailyReviewProgress", map[string]any{"chunk": "x"})
	SafeEventsEmit(nil, "dailyReviewGenerated", map[string]any{"date": "2026-10-08"})
	SafeEventsEmit(context.TODO(), "promptBacktestProgress", map[string]any{"progress": 1})
}

// TestEmitStockDataChangedCtxUnset 验证 appCtx 未设置（Web 端 / 测试环境）时
// EmitStockDataChanged 不崩溃。
func TestEmitStockDataChangedCtxUnset(t *testing.T) {
	EmitStockDataChanged()
}
