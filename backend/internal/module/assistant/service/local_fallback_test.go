package service

import (
	"context"
	"strings"
	"testing"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	assistantdto "github.com/han139242/course-recommend-agents/backend/internal/module/assistant/dto"
)

func TestChatPublicFallsBackLocallyWithoutAIConfig(t *testing.T) {
	t.Parallel()

	service := New(config.AIConfig{}, nil, nil, nil, nil, nil, nil, nil, nil)
	response, err := service.ChatPublic(context.Background(), assistantdto.ChatRequest{
		Messages: []map[string]any{
			{
				"role": "system",
				"content": strings.Join([]string{
					"你是知识图谱学习助手，请用中文回答。",
					"当前知识点：Transformer",
					"课程：深度学习",
					"简介：Transformer 是一种基于注意力机制的序列建模架构。",
					"前置知识：注意力机制、序列建模",
					"掌握度：0",
				}, "\n"),
			},
			{"role": "user", "content": "帮我解释 Transformer"},
		},
	})
	if err != nil {
		t.Fatalf("expected local fallback response, got error: %v", err)
	}
	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if !strings.Contains(response.Content, "Transformer") {
		t.Fatalf("expected fallback reply to mention knowledge name, got %q", response.Content)
	}
	if len(response.Actions) != 0 {
		t.Fatalf("expected no actions for explanation-only fallback, got %d", len(response.Actions))
	}
}

func TestChatPublicKeepsGraphActionsWithoutAIConfig(t *testing.T) {
	t.Parallel()

	service := New(config.AIConfig{}, nil, nil, nil, nil, nil, nil, nil, nil)
	response, err := service.ChatPublic(context.Background(), assistantdto.ChatRequest{
		Messages: []map[string]any{
			{"role": "user", "content": "展开 Transformer 的关联节点并聚焦到图里"},
		},
	})
	if err != nil {
		t.Fatalf("expected action fallback response, got error: %v", err)
	}
	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if len(response.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(response.Actions))
	}
	if response.Actions[0].Type != "expand_knowledge" || response.Actions[0].Target != "Transformer" {
		t.Fatalf("unexpected first action: %#v", response.Actions[0])
	}
	if response.Actions[1].Type != "focus_knowledge" || response.Actions[1].Target != "Transformer" {
		t.Fatalf("unexpected second action: %#v", response.Actions[1])
	}
}
