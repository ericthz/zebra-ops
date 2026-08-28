package memory

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestGetSimpleMemory_ReturnsSameInstance(t *testing.T) {
	a := GetSimpleMemory("id-1")
	b := GetSimpleMemory("id-1")
	if a != b {
		t.Fatal("GetSimpleMemory should return the same instance for the same id")
	}
}

func TestGetSimpleMemory_DifferentIDs(t *testing.T) {
	a := GetSimpleMemory("id-a")
	b := GetSimpleMemory("id-b")
	if a == b {
		t.Fatal("GetSimpleMemory should return different instances for different ids")
	}
}

func TestSimpleMemory_SetAndGet(t *testing.T) {
	m := GetSimpleMemory("t-set-get")
	m.Messages = []*schema.Message{}

	m.SetMessages(schema.UserMessage("你好"))
	m.SetMessages(schema.SystemMessage("你好，有什么可以帮你？"))

	msgs := m.GetMessages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != schema.User || msgs[0].Content != "你好" {
		t.Fatalf("unexpected first message: %+v", msgs[0])
	}
	if msgs[1].Role != schema.System || msgs[1].Content != "你好，有什么可以帮你？" {
		t.Fatalf("unexpected second message: %+v", msgs[1])
	}
}

func TestSimpleMemory_WindowTrimming(t *testing.T) {
	m := GetSimpleMemory("t-window")
	m.Messages = []*schema.Message{}

	// MaxWindowSize = 6，写入 8 条（4 组问答），应裁剪到 6 条
	for i := 0; i < 4; i++ {
		m.SetMessages(schema.UserMessage("q" + string(rune('0'+i))))
		m.SetMessages(schema.SystemMessage("a" + string(rune('0'+i))))
	}

	msgs := m.GetMessages()
	if len(msgs) != 6 {
		t.Fatalf("expected 6 messages after trimming, got %d", len(msgs))
	}
	// 裁剪后应保留最后一组问答（丢弃最旧的 1 组）
	if msgs[0].Content != "q1" {
		t.Fatalf("expected window to drop oldest pair, first message got %q", msgs[0].Content)
	}
}

func TestSimpleMemory_WindowTrimmingKeepsPairing(t *testing.T) {
	m := GetSimpleMemory("t-pairing")
	m.Messages = []*schema.Message{}

	// 正常场景：成对追加（4 组问答 = 8 条），超窗后应裁剪到 6 条且保持配对
	for i := 0; i < 4; i++ {
		m.SetMessages(schema.UserMessage("q" + string(rune('0'+i))))
		m.SetMessages(schema.SystemMessage("a" + string(rune('0'+i))))
	}

	msgs := m.GetMessages()
	if len(msgs) != 6 {
		t.Fatalf("expected 6 messages, got %d", len(msgs))
	}
	// 应丢弃最旧的一组（q0/a0），保留 q1..a3，首尾配对完整
	if msgs[0].Content != "q1" || msgs[0].Role != schema.User {
		t.Fatalf("expected window to drop oldest pair, first message got %+v", msgs[0])
	}
	if msgs[5].Content != "a3" || msgs[5].Role != schema.System {
		t.Fatalf("expected window to end with last assistant reply, got %+v", msgs[5])
	}
}

func TestSimpleMemory_OddOverflowDropsEvenCount(t *testing.T) {
	m := GetSimpleMemory("t-odd")
	m.Messages = []*schema.Message{}

	// 边界：7 条（奇数）超窗，应丢弃偶数条（2 条）保持 5 条
	for i := 0; i < 7; i++ {
		m.SetMessages(schema.UserMessage("m" + string(rune('0'+i))))
	}

	msgs := m.GetMessages()
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages after dropping even count, got %d", len(msgs))
	}
}
