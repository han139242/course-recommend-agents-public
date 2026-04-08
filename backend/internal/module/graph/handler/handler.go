package handler

import (
	"github.com/gin-gonic/gin"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供图谱查询 HTTP 处理能力。
type Handler struct {
	service *graphservice.Service
}

// New 创建图谱处理器。
func New(service *graphservice.Service) *Handler {
	return &Handler{service: service}
}

// GetSubgraph 查询知识子图。
func (h *Handler) GetSubgraph(ctx *gin.Context) {
	var query graphdto.SubgraphQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.GetSubgraph(ctx.Request.Context(), query)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// GetKnowledgeDetail 查询知识点详情。
func (h *Handler) GetKnowledgeDetail(ctx *gin.Context) {
	response, err := h.service.GetKnowledgeDetail(ctx.Request.Context(), ctx.Param("knowledgeId"))
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
