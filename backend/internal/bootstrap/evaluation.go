package bootstrap

import (
	evaluationmodule "github.com/han139242/course-recommend-agents/backend/internal/module/evaluation"
	evaluationservice "github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/service"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
)

// NewEvaluationModule 创建评测模块并注入 KT-KGPath 评分器。
func NewEvaluationModule(scorer *recommendservice.Scorer) *evaluationmodule.Module {
	svc := evaluationservice.New(scorer)
	return evaluationmodule.New(svc)
}
