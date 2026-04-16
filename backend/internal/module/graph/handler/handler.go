package handler

import (
	"github.com/gin-gonic/gin"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphservice "github.com/han139242/course-recommend-agents/backend/internal/module/graph/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler provides HTTP handlers for graph queries.
type Handler struct {
	service *graphservice.Service
}

// New creates a graph handler.
func New(service *graphservice.Service) *Handler {
	return &Handler{service: service}
}

// GetSubgraph returns a one-hop graph around a root node.
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

// GetFullGraph returns a full knowledge graph snapshot for the homepage.
func (h *Handler) GetFullGraph(ctx *gin.Context) {
	var query graphdto.FullGraphQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	response, err := h.service.GetFullGraph(ctx.Request.Context(), query)
	if err != nil {
		writeError(ctx, err)
		return
	}

	result.Success(ctx, response)
}

// GetKnowledgeDetail returns detail information for a knowledge node.
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
