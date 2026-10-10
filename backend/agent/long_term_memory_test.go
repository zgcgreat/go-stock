package agent

import (
	"errors"
	"strings"
	"testing"
)

// 回归背景（2026-10-08）：无 ModelType=embedding 配置时自动回退选中 DeepSeek 官方 API，
// 而 DeepSeek 官方无 /embeddings 端点，导致每次归档报告都 404（写入 chunk 0/29 失败）。
// 熔断机制要求：404/401 后标记不可用、后续写入/检索跳过、reset 后清除。

func TestMarkIfEmbeddingFailure404(t *testing.T) {
	t.Cleanup(func() {
		embeddingFailMu.Lock()
		embeddingFailedCfgIDs = map[uint]string{}
		embeddingFailMu.Unlock()
		setCurrentEmbeddingCfgID(0)
	})

	setCurrentEmbeddingCfgID(1)
	// 模拟 chromem-go 的真实错误文本（DeepSeek 404 场景）
	err404 := errors.New("写入 chunk 0/29 失败: couldn't create embedding of document: " +
		"error response from the embedding API: 404 Not Found")
	markIfEmbeddingFailure(err404)

	reason, broken := embeddingUnavailableReason(1)
	if !broken {
		t.Fatal("404 embedding 错误应触发熔断标记")
	}
	if !strings.Contains(reason, "404") {
		t.Fatalf("熔断原因应保留原始错误，got: %s", reason)
	}

	// 当前绑定的服务应被视为已熔断（写入/检索路径据此跳过）
	if _, broken := currentEmbeddingBroken(); !broken {
		t.Fatal("currentEmbeddingBroken 应返回 true")
	}
}

func TestMarkIfEmbeddingFailureTransient(t *testing.T) {
	t.Cleanup(func() {
		embeddingFailMu.Lock()
		embeddingFailedCfgIDs = map[uint]string{}
		embeddingFailMu.Unlock()
		setCurrentEmbeddingCfgID(0)
	})

	setCurrentEmbeddingCfgID(2)
	// 429 限流 / 网络抖动属瞬态错误，不应熔断
	transient := errors.New("couldn't create embedding of document: " +
		"error response from the embedding API: 429 Too Many Requests")
	markIfEmbeddingFailure(transient)
	if _, broken := embeddingUnavailableReason(2); broken {
		t.Fatal("429 瞬态错误不应触发熔断")
	}

	// 非 embedding 错误（如 DB 写盘失败）不应熔断
	other := errors.New("write to disk failed: permission denied")
	markIfEmbeddingFailure(other)
	if _, broken := embeddingUnavailableReason(2); broken {
		t.Fatal("非 embedding 错误不应触发熔断")
	}
}

func TestResetClearsEmbeddingFailures(t *testing.T) {
	setCurrentEmbeddingCfgID(3)
	markEmbeddingUnavailable(3, "404 Not Found")
	if _, broken := embeddingUnavailableReason(3); !broken {
		t.Fatal("前置：熔断标记应存在")
	}

	resetLongTermMemoryStore()

	if _, broken := embeddingUnavailableReason(3); broken {
		t.Fatal("resetLongTermMemoryStore 应清除熔断标记（用户切换向量服务后允许重试）")
	}
	if _, broken := currentEmbeddingBroken(); broken {
		t.Fatal("reset 后当前绑定应视为未熔断")
	}
}

func TestParseEmbeddingCfgID(t *testing.T) {
	tests := []struct {
		summary string
		want    uint
	}{
		{"aiConfig#1:https://api.deepseek.com=text-embedding-3-small", 1},
		{"aiConfig#42:https://api.siliconflow.cn=bge-m3", 42},
		{"", 0},
		{"garbage", 0},
		{"aiConfig#abc:url=model", 0},
	}
	for _, tt := range tests {
		if got := parseEmbeddingCfgID(tt.summary); got != tt.want {
			t.Errorf("parseEmbeddingCfgID(%q) = %d, want %d", tt.summary, got, tt.want)
		}
	}
}

func TestMarkEmbeddingUnavailableZeroID(t *testing.T) {
	// cfgID=0（未知绑定）不应写入熔断表，避免误伤
	markEmbeddingUnavailable(0, "404")
	if _, broken := embeddingUnavailableReason(0); broken {
		t.Fatal("cfgID=0 不应被熔断标记")
	}
}
