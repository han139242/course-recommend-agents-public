package service

import (
	"context"

	"github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/entity"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
)

// Service 评测服务。
type Service struct {
	registry *BaselineRegistry
}

// New 创建评测服务，注入 KT-KGPath 评分器用于构建 baseline。
func New(scorer *recommendservice.Scorer) *Service {
	registry := NewBaselineRegistry()
	registry.Register("kt_kgpath", KTKGPathBaseline(scorer))
	return &Service{registry: registry}
}

// RunBenchmark 运行所有 baseline 的评测并返回聚合结果。
func (s *Service) RunBenchmark(ctx context.Context, tasks []entity.BenchmarkTask) []entity.BenchmarkResult {
	results := make([]entity.BenchmarkResult, 0, len(s.registry.List()))

	for _, name := range s.registry.List() {
		agg := aggregateResult{name: name}
		for _, task := range tasks {
			ids := s.registry.Run(name, task)
			if ids == nil {
				continue
			}
			agg.record(ids, task)
		}
		results = append(results, agg.finalize())
	}
	return results
}

type aggregateResult struct {
	name                   string
	hit1, hit3, hit5       float64
	ndcg5, ndcg10          float64
	mrr                    float64
	prereq                 float64
	smoothness             float64
	pathLen                float64
	coverage               float64
	count                  int
}

func (a *aggregateResult) record(ids []string, task entity.BenchmarkTask) {
	a.count++
	a.hit1 += ComputeHitAtK(ids, task.GroundTruth, 1)
	a.hit3 += ComputeHitAtK(ids, task.GroundTruth, 3)
	a.hit5 += ComputeHitAtK(ids, task.GroundTruth, 5)
	a.ndcg5 += ComputeNDCGAtK(ids, task.GroundTruth, 5)
	a.ndcg10 += ComputeNDCGAtK(ids, task.GroundTruth, 10)
	a.mrr += ComputeMRR(ids, task.GroundTruth)
	a.prereq += ComputePrerequisiteViolationRate(ids, task.Prerequisites)
	a.smoothness += ComputeDifficultySmoothness(ids, task.Difficulties)
	a.pathLen += float64(len(ids))
	if len(task.CandidatePool) > 0 {
		a.coverage += float64(len(ids)) / float64(len(task.CandidatePool))
	}
}

func (a *aggregateResult) finalize() entity.BenchmarkResult {
	if a.count == 0 {
		return entity.BenchmarkResult{Baseline: a.name}
	}
	n := float64(a.count)
	return entity.BenchmarkResult{
		Baseline:              a.name,
		HitAt1:                round(a.hit1/n, 4),
		HitAt3:                round(a.hit3/n, 4),
		HitAt5:                round(a.hit5/n, 4),
		NDCGAt5:               round(a.ndcg5/n, 4),
		NDCGAt10:              round(a.ndcg10/n, 4),
		MRR:                   round(a.mrr/n, 4),
		PrerequisiteViolation: round(a.prereq/n, 4),
		DifficultySmoothness:  round(a.smoothness/n, 4),
		AveragePathLength:     round(a.pathLen/n, 2),
		CoverageRate:          round(a.coverage/n, 4),
	}
}

func round(v float64, decimals int) float64 {
	scale := 1.0
	for i := 0; i < decimals; i++ {
		scale *= 10
	}
	return float64(int(v*scale+0.5)) / scale
}

// BuildTasks 从学生数据构建评测任务列表（供外部调用）。
func BuildTasks(records []entity.AnswerRecord, candidates []entity.CandidateNode, prereqs []entity.PrerequisiteEdge, difficulties map[string]float64, groundTruth []string) []entity.BenchmarkTask {
	if len(records) == 0 || len(candidates) == 0 {
		return nil
	}
	return []entity.BenchmarkTask{{
		Name:           "default",
		StudentHistory: records,
		CandidatePool:  candidates,
		GroundTruth:    groundTruth,
		Prerequisites:  prereqs,
		Difficulties:   difficulties,
	}}
}

// RegisterExtra 允许在初始化后注册额外 baseline。
func (s *Service) RegisterExtra(name string, fn BaselineFunc) {
	s.registry.Register(name, fn)
}
