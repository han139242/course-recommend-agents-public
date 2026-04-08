package handler

import (
	"github.com/gin-gonic/gin"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供推荐路径 HTTP 处理能力。
type Handler struct {
	service *recommendservice.Service
}

// New 创建推荐处理器。
func New(service *recommendservice.Service) *Handler {
	return &Handler{service: service}
}

// GetPath 查询推荐路径。
func (h *Handler) GetPath(ctx *gin.Context) {
	var query recommenddto.PathQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.GetPath(ctx.Request.Context(), query)
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
