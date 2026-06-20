package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	resourcegendto "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/dto"
	resourcegenservice "github.com/han139242/course-recommend-agents/backend/internal/module/resourcegen/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

type Handler struct {
	service *resourcegenservice.Service
}

func New(service *resourcegenservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Generate(ctx *gin.Context) {
	var request resourcegendto.GenerateResourcesRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.Generate(ctx.Request.Context(), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetJob(ctx *gin.Context) {
	response, err := h.service.GetJob(ctx.Request.Context(), ctx.Param("jobId"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// StreamJob streams generation job progress over SSE so the frontend gets live
// agent-step and progress updates without polling. Each event is the JSON job
// view; the stream ends with a [DONE] sentinel once the job completes or fails.
func (h *Handler) StreamJob(ctx *gin.Context) {
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	flusher, canFlush := ctx.Writer.(http.Flusher)
	reqCtx := ctx.Request.Context()
	jobID := ctx.Param("jobId")

	writeEvent := func(payload any) {
		data, _ := json.Marshal(payload)
		_, _ = ctx.Writer.Write([]byte("data: "))
		_, _ = ctx.Writer.Write(data)
		_, _ = ctx.Writer.Write([]byte("\n\n"))
		if canFlush {
			flusher.Flush()
		}
	}

	finish := func() {
		_, _ = ctx.Writer.Write([]byte("data: [DONE]\n\n"))
		if canFlush {
			flusher.Flush()
		}
	}

	// emit 推送一次任务快照，返回任务是否已结束。
	emit := func() bool {
		view, err := h.service.GetJob(reqCtx, jobID)
		if err != nil {
			appErr := apperr.AsAppError(err)
			writeEvent(map[string]any{"error": appErr.Message})
			return true
		}
		writeEvent(view)
		return view.Status == "completed" || view.Status == "failed"
	}

	if emit() {
		finish()
		return
	}

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(3 * time.Minute)

	for {
		select {
		case <-reqCtx.Done():
			return
		case <-ticker.C:
			if emit() || time.Now().After(deadline) {
				finish()
				return
			}
		}
	}
}

func (h *Handler) ExportJob(ctx *gin.Context) {
	response, err := h.service.ExportJobMarkdown(ctx.Request.Context(), ctx.Param("jobId"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetPresentationOutline(ctx *gin.Context) {
	response, err := h.service.GetPresentationOutline(ctx.Request.Context(), ctx.Param("jobId"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetDemoScript(ctx *gin.Context) {
	response, err := h.service.GetDemoScript(ctx.Request.Context(), ctx.Param("jobId"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) ListResources(ctx *gin.Context) {
	var query resourcegendto.ResourceQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.ListResources(ctx.Request.Context(), query)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetResource(ctx *gin.Context) {
	response, err := h.service.GetResource(ctx.Request.Context(), ctx.Param("resourceId"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetSafetyReport(ctx *gin.Context) {
	response, err := h.service.GetSafetyReport(ctx.Request.Context(), ctx.Param("resourceId"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) SubmitFeedback(ctx *gin.Context) {
	var request resourcegendto.SubmitFeedbackRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.SubmitFeedback(ctx.Request.Context(), ctx.Param("resourceId"), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetRecommendations(ctx *gin.Context) {
	response, err := h.service.GetRecommendations(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetEffectSummary(ctx *gin.Context) {
	response, err := h.service.GetEffectSummary(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GenerateQuiz(ctx *gin.Context) {
	var request resourcegendto.GenerateQuizRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.GenerateQuiz(ctx.Request.Context(), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetFeed(ctx *gin.Context) {
	response, err := h.service.GetFeed(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) RecordLearningEvent(ctx *gin.Context) {
	var request resourcegendto.RecordLearningEventRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.RecordLearningEvent(ctx.Request.Context(), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func writeError(ctx *gin.Context, err error) {
	appErr := apperr.AsAppError(err)
	result.Error(ctx, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
