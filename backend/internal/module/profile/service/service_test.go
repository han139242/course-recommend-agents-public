package service

import (
	"strings"
	"testing"

	profiledto "github.com/han139242/course-recommend-agents/backend/internal/module/profile/dto"
	profileentity "github.com/han139242/course-recommend-agents/backend/internal/module/profile/entity"
)

func TestInferProfileDimensionsAlwaysBuildsSixDimensions(t *testing.T) {
	dimensions := inferProfileDimensions("我是大二人工智能学生，机器学习基础薄弱，喜欢代码案例和导图，希望完成智能体项目。")
	if len(dimensions) != 6 {
		t.Fatalf("expected 6 dimensions, got %d", len(dimensions))
	}
	seen := map[string]bool{}
	for _, dimension := range dimensions {
		if dimension.Key == "" || dimension.Value == "" || dimension.Evidence == "" {
			t.Fatalf("dimension is incomplete: %#v", dimension)
		}
		seen[dimension.Key] = true
	}
	for _, required := range requiredProfileDimensionNames {
		if !seen[required.Key] {
			t.Fatalf("missing required dimension %s", required.Key)
		}
	}
}

func TestNormalizeAIDimensionsRejectsIncompleteAndClampsConfidence(t *testing.T) {
	input := make([]profiledto.ProfileDimensionView, 0, 6)
	for _, required := range requiredProfileDimensionNames {
		input = append(input, profiledto.ProfileDimensionView{Key: required.Key, Value: "value", Confidence: 1.5})
	}
	normalized := normalizeAIDimensions(input)
	if len(normalized) != 6 || normalized[0].Confidence != 1 {
		t.Fatalf("unexpected normalized dimensions: %#v", normalized)
	}
	if got := normalizeAIDimensions(input[:5]); got != nil {
		t.Fatalf("expected incomplete model output to be rejected, got %#v", got)
	}
}

func TestRefreshDimensionsFromEventsUsesPracticeAndResourceSignals(t *testing.T) {
	current := inferProfileDimensions("想学习智能体")
	events := []profileentity.LearningEvent{
		{EventType: "practice_answered", KnowledgeName: "工具调用", ResourceType: "coding_case", PayloadJSON: `{"correct":false}`},
		{EventType: "resource_completed", KnowledgeName: "工具调用", ResourceType: "coding_case", PayloadJSON: `{}`},
	}
	next, insights := refreshDimensionsFromEvents(current, events)
	if len(insights) < 2 {
		t.Fatalf("expected refresh insights, got %#v", insights)
	}
	values := map[string]string{}
	for _, dimension := range next {
		values[dimension.Key] = dimension.Value
	}
	if !strings.Contains(values["learning_progress"], "完成") {
		t.Fatalf("learning progress was not refreshed: %s", values["learning_progress"])
	}
	if !strings.Contains(values["error_preference"], "易错") {
		t.Fatalf("error preference was not refreshed: %s", values["error_preference"])
	}
}
