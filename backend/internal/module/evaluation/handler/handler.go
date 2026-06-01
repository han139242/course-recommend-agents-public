package handler

import (
	"github.com/gin-gonic/gin"
	evdto "github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/dto"
	evsvc "github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供评测 HTTP 处理能力。
type Handler struct {
	service *evsvc.Service
}

// New 创建评测处理器。
func New(service *evsvc.Service) *Handler {
	return &Handler{service: service}
}

// RunBenchmark 执行评测并返回各 baseline 结果。
func (h *Handler) RunBenchmark(ctx *gin.Context) {
	var query evdto.RunBenchmarkQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		query.TaskName = "default"
	}

	tasks := evsvc.BuildSampleTasks()
	if len(tasks) == 0 {
		result.Success(ctx, evdto.RunBenchmarkResponse{
			TaskName: query.TaskName,
			Results:  nil,
		})
		return
	}

	benchResults := h.service.RunBenchmark(ctx.Request.Context(), tasks)
	result.Success(ctx, evdto.RunBenchmarkResponse{
		TaskName: query.TaskName,
		Results:  benchResults,
	})
}
