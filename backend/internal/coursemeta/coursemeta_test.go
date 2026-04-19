package coursemeta

import "testing"

func TestResolveNodeCourseTitlePrefersCuratedSource(t *testing.T) {
	title := ResolveNodeCourseTitle("course_ai_intro", "curated/course_ai_intro", "", "", "", "人工智能基础知识骨架")
	if title != "人工智能导论" {
		t.Fatalf("expected normalized course title, got %q", title)
	}
}

func TestNormalizeCourseTitleUsesLegacyAlias(t *testing.T) {
	title := NormalizeCourseTitle("人工智能基础知识骨架")
	if title != "人工智能导论" {
		t.Fatalf("expected alias to normalize, got %q", title)
	}
}

func TestNormalizeCourseTitleUsesKeywordHints(t *testing.T) {
	title := NormalizeCourseTitleWithFallback("batch_22_automl_compression_edge_ai", "")
	if title != "AI工程" {
		t.Fatalf("expected keyword hint to resolve AI工程, got %q", title)
	}
}
