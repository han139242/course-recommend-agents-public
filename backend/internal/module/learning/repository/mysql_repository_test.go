package repository

import (
	"testing"

	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
)

// TestContainsOption 验证选项校验只接受题目定义中出现过的选项键。
func TestContainsOption(t *testing.T) {
	if !containsOption([]learningentity.QuestionOption{{Key: "A", Text: "选项A"}}, "A") {
		t.Fatal("expected option to exist")
	}
	if containsOption([]learningentity.QuestionOption{{Key: "A", Text: "选项A"}}, "B") {
		t.Fatal("expected option to be absent")
	}
}
