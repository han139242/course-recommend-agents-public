package service

import (
	"strings"
	"testing"

	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	resourcegenentity "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/entity"
)

func TestNormalizeResourceTypesDefaultsAndDeduplicates(t *testing.T) {
	defaults := normalizeResourceTypes(nil)
	if len(defaults) < 5 {
		t.Fatalf("expected at least five default resource types, got %#v", defaults)
	}
	got := normalizeResourceTypes([]string{"mind_map", "mind_map", "invalid", "coding_case"})
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	if len(got) != 2 || !seen["mind_map"] || !seen["coding_case"] {
		t.Fatalf("unexpected normalized resource types: %#v", got)
	}
}

func TestAppendGroundingEvidenceUsesGraphSource(t *testing.T) {
	ctx := generationContext{
		KnowledgeID:   "kg-1",
		KnowledgeName: "工具调用",
		KnowledgeDetail: &graphdto.KnowledgeDetailView{
			ID:            "kg-1",
			Name:          "工具调用",
			CuratedSource: "《人工智能导论》第 6 章",
			Prerequisites: []string{"函数调用"},
		},
	}
	content := appendGroundingEvidence("explanation_doc", "# 工具调用", ctx)
	for _, expected := range []string{"## 依据与核验", "《人工智能导论》第 6 章", "函数调用"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("grounding appendix missing %q: %s", expected, content)
		}
	}
	if got := appendGroundingEvidence("mind_map", "mindmap\n  root", ctx); strings.Contains(got, "依据与核验") {
		t.Fatalf("mermaid content must remain syntax-only: %s", got)
	}
}

func TestSafetyReportRequiresGroundingAndBlocksHighRiskContent(t *testing.T) {
	safe := resourcegenentity.GeneratedResource{ID: 1, ResourceType: "explanation_doc", Format: "markdown", Reason: "画像需要", Content: "# 讲解\n正文\n## 依据与核验\n- 来源：课程知识图谱"}
	if report := buildSafetyReport(safe); report.Status != "passed" {
		t.Fatalf("expected grounded resource to pass, got %#v", report)
	}
	risky := resourcegenentity.GeneratedResource{ID: 2, ResourceType: "coding_case", Format: "markdown", Content: "保证 100% 绝对正确，这是唯一答案，可用于攻击和绕过。https://fake.invalid"}
	if report := buildSafetyReport(risky); report.Status != "blocked" {
		t.Fatalf("expected risky ungrounded resource to be blocked, got %#v", report)
	}
}

func TestBuildMediaStatusRecognizesReadyAndPending(t *testing.T) {
	pending := resourcegenentity.GeneratedResource{ID: 3, Format: "video", Content: `{"kind":"video","status":"queued","providerTaskId":"task-1"}`}
	if status := buildMediaStatus(pending); status.Ready || status.ProviderTaskID != "task-1" {
		t.Fatalf("unexpected pending status: %#v", status)
	}
	ready := resourcegenentity.GeneratedResource{ID: 4, Format: "image", Content: `{"kind":"image","url":"https://cdn.example/image.png","status":"completed"}`}
	if status := buildMediaStatus(ready); !status.Ready || status.URL == "" {
		t.Fatalf("unexpected ready status: %#v", status)
	}
}
