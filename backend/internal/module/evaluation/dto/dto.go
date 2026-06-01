package dto

import "github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/entity"

// RunBenchmarkQuery 定义评测请求参数。
type RunBenchmarkQuery struct {
	TaskName string `form:"task" binding:"omitempty,max=128"`
}

// RunBenchmarkResponse 评测响应。
type RunBenchmarkResponse struct {
	TaskName string                   `json:"taskName"`
	Results  []entity.BenchmarkResult `json:"results"`
}
