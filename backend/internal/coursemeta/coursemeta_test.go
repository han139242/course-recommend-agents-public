package coursemeta

import "testing"

func TestResolveNodeCourseTitlePrefersCuratedSource(t *testing.T) {
	title := ResolveNodeCourseTitle("course_ai_intro", "curated/course_ai_intro", "", "", "", "人工智能基础知识骨架")
	if title != "人工智能导论" {
		t.Fatalf("expected normalized course title, got %q", title)
	}
}

func TestNormalizeCourseTitleDoesNotCollapseGenericSkeletonLabels(t *testing.T) {
	if title := NormalizeCourseTitle("人工智能基础知识骨架"); title != DefaultCourseTitle {
		t.Fatalf("expected generic skeleton title to fall back to %q, got %q", DefaultCourseTitle, title)
	}
	if title := NormalizeCourseTitle("人工智能课程知识图谱"); title != DefaultCourseTitle {
		t.Fatalf("expected generic graph title to fall back to %q, got %q", DefaultCourseTitle, title)
	}
}

func TestNormalizeCourseTitleUsesKeywordHints(t *testing.T) {
	title := NormalizeCourseTitleWithFallback("batch_22_automl_compression_edge_ai", "")
	if title != "AI工程" {
		t.Fatalf("expected keyword hint to resolve AI工程, got %q", title)
	}
}

func TestResolveNodeCourseTitleFromHintsReclassifiesGenericOntologyNodesByKnowledgeName(t *testing.T) {
	if title := ResolveNodeCourseTitleFromHints("curated_ai_professional_ontology", "curated/curated_ai_professional_ontology", "", "", "", "人工智能基础知识骨架", "图像分割", "图像分割", ""); title != "计算机视觉" {
		t.Fatalf("expected 图像分割 to resolve to 计算机视觉, got %q", title)
	}
	if title := ResolveNodeCourseTitleFromHints("curated_ai_professional_ontology", "curated/curated_ai_professional_ontology", "", "", "", "人工智能基础知识骨架", "马尔可夫决策过程", "马尔可夫决策过程", ""); title != "强化学习" {
		t.Fatalf("expected 马尔可夫决策过程 to resolve to 强化学习, got %q", title)
	}
	if title := ResolveNodeCourseTitleFromHints("curated_ai_professional_ontology", "curated/curated_ai_professional_ontology", "", "", "", "人工智能基础知识骨架", "知识表示", "知识表示", ""); title != "人工智能导论" {
		t.Fatalf("expected 知识表示 to resolve to 人工智能导论, got %q", title)
	}
}
