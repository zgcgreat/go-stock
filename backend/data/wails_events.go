package data

import (
	"context"
	"go-stock/backend/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// -----------------------------------------------------------------------------------
// Wails 事件安全推送
//
// 背景（真实崩溃）：Web 端手动生成复盘报告时，handler 传入 context.Background()，
// 进度推送经 progressEmitter → safeEventsEmit → runtime.EventsEmit。wails 的
// getEvents(ctx) 在 ctx 中找不到 "events" 键时走的是 log.Fatalf（os.Exit(1)），
// **不是 panic**，因此外层的 recover() 完全无效，进程被直接杀死（exit status 1）。
//
// 结论：唯一可靠的防护是调用前校验 ctx 是否携带 wails 生命周期 ctx 的 events 句柄，
// 不能依赖 recover。桌面端 startup 由 SetAppCtx 注入真实 ctx，推送正常；
// Web 端没有 wails 事件通道，跳过推送即为正确降级。
// -----------------------------------------------------------------------------------

// wailsEventsCtxKey 与 wails 内部 runtime.getEvents 读取的键一致。
// 该键为 wails 私有约定（未导出），只能按字面量读取；此处只做存在性探测，不写入。
const wailsEventsCtxKey = "events"

// HasWailsEvents 判断 ctx 是否携带 wails 生命周期 ctx 的事件句柄。
// 桌面端 startup 注入的 ctx 返回 true；context.Background()/gin 请求 ctx 返回 false。
func HasWailsEvents(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	return ctx.Value(wailsEventsCtxKey) != nil
}

// SafeEventsEmit 安全推送 Wails 事件：ctx 不携带 events 句柄时（Web 端、单元测试、
// context.Background()）直接跳过，不进入 runtime.EventsEmit，避免 log.Fatalf 杀进程。
//
// recover 仅作为第二道防线保留：即便未来 ctx 结构变化导致运行时异常，也不会带崩进程。
func SafeEventsEmit(ctx context.Context, event string, payload any) {
	if !HasWailsEvents(ctx) {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.SugaredLogger.Warnf("emit event %s panic recovered: %v", event, r)
			}
		}()
		runtime.EventsEmit(ctx, event, payload)
	}()
}
