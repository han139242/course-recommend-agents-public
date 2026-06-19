package handler

import (
	"github.com/gin-gonic/gin"
	profiledto "github.com/han139242/course-recommend-agents/backend/internal/module/profile/dto"
	profileservice "github.com/han139242/course-recommend-agents/backend/internal/module/profile/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

type Handler struct {
	service *profileservice.Service
}

func New(service *profileservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ExtractProfile(ctx *gin.Context) {
	var request profiledto.ExtractProfileRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.ExtractProfile(ctx.Request.Context(), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GetCurrent(ctx *gin.Context) {
	response, err := h.service.GetCurrent(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) UpsertProfile(ctx *gin.Context) {
	var request profiledto.UpsertProfileRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.UpsertProfile(ctx.Request.Context(), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) RefreshProfile(ctx *gin.Context) {
	response, err := h.service.GetCurrent(ctx.Request.Context())
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
