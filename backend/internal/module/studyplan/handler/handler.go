package handler

import (
	"github.com/gin-gonic/gin"
	studyplandto "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/dto"
	studyplanservice "github.com/han139242/course-recommend-agents/backend/internal/module/studyplan/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

type Handler struct {
	service *studyplanservice.Service
}

func New(service *studyplanservice.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetProfile(ctx *gin.Context) {
	response, err := h.service.GetProfile(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) UpsertProfile(ctx *gin.Context) {
	var request studyplandto.GoalProfileRequest
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

func (h *Handler) GetCurrentPlan(ctx *gin.Context) {
	response, err := h.service.GetCurrentPlan(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) GeneratePlan(ctx *gin.Context) {
	response, err := h.service.GeneratePlan(ctx.Request.Context())
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func (h *Handler) AdjustPlan(ctx *gin.Context) {
	response, err := h.service.AdjustPlan(ctx.Request.Context())
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
