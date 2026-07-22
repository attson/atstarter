package docker

import (
	"context"
	"testing"
	"time"
)

// TestExecDeadlineRespectsCaller 验证:调用者传入的 deadline 会被原样采用,
// 不会被兜底超时覆盖成更短的 10s。这是修复 compose 生命周期命令(可能几分钟)
// 被 10s 兜底误杀的核心断言。
func TestExecDeadlineRespectsCaller(t *testing.T) {
	// 调用者要一个 5 分钟的 deadline(模拟 compose up 拉镜像)。
	want := time.Now().Add(5 * time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), want)
	defer cancel()

	got := execDeadline(ctx)
	// execDeadline 应把调用者的 deadline 原样返回,不缩短成 ~10s。
	if got.Before(time.Now().Add(4 * time.Minute)) {
		t.Fatalf("execDeadline 把调用者的 5min deadline 缩短了:got %v", got)
	}
}

// TestExecDeadlineFallback 验证:无 deadline 的 ctx 会拿到一个兜底 deadline,
// 且兜底是分钟级(修复方案要求 5 分钟),而非旧的 10s。
func TestExecDeadlineFallback(t *testing.T) {
	got := execDeadline(context.Background())
	// 兜底应远大于旧的 10s(至少 1 分钟以上)。
	if got.Before(time.Now().Add(1 * time.Minute)) {
		t.Fatalf("兜底 deadline 太短(疑似仍是 10s):got %v", got)
	}
}

// TestDefaultExecRespectsShortDeadline 端到端验证:快命令传入短 deadline 会按其超时。
func TestDefaultExecRespectsShortDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	res := defaultExec(ctx, "sleep", "5")
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("defaultExec 耗时 %v,50ms deadline 未生效", elapsed)
	}
	if res.Err == nil && res.ExitCode == 0 {
		t.Errorf("期望 sleep 被中断,但 res = %+v", res)
	}
}
