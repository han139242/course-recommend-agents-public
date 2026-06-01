package handler

import (
	"github.com/gin-gonic/gin"
	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
	frontierservice "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供前沿扩展 HTTP 处理能力。
type Handler struct {
	service *frontierservice.Service
}

// New 创建前沿扩展处理器。
func New(service *frontierservice.Service) *Handler {
	return &Handler{service: service}
}

// GetBriefing 查询前沿扩展摘要。
func (h *Handler) GetBriefing(ctx *gin.Context) {
	var query frontierdto.BriefingQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.GetBriefing(ctx.Request.Context(), query)
	if err != nil {
		appErr := apperr.AsAppError(err)
		result.Error(ctx, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	result.Success(ctx, response)
}

// ListLibrary 查询前沿资料中心列表。
func (h *Handler) ListLibrary(ctx *gin.Context) {
	var query frontierdto.LibraryQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.ListLibrary(ctx.Request.Context(), query)
	if err != nil {
		appErr := apperr.AsAppError(err)
		result.Error(ctx, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	result.Success(ctx, response)
}
