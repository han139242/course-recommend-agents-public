package handler

import (
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

func (h *Handler) ExportJob(ctx *gin.Context) {
	response, err := h.service.ExportJobMarkdown(ctx.Request.Context(), ctx.Param("jobId"))
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

func writeError(ctx *gin.Context, err error) {
	appErr := apperr.AsAppError(err)
	result.Error(ctx, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
