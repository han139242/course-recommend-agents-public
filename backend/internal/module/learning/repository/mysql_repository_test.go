package repository

import (
	"testing"

	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
)

func TestContainsOption(t *testing.T) {
	if !containsOption([]learningentity.QuestionOption{{Key: "A", Text: "选项A"}}, "A") {
		t.Fatal("expected option to exist")
	}
	if containsOption([]learningentity.QuestionOption{{Key: "A", Text: "选项A"}}, "B") {
		t.Fatal("expected option to be absent")
	}
}
