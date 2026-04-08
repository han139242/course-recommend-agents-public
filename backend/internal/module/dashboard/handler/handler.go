package handler

import (
	"github.com/gin-gonic/gin"
	authentity "github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	dashboardservice "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供仪表盘 HTTP 处理能力。
type Handler struct {
	service     *dashboardservice.Service
	authService *authservice.AuthService
}

// New 创建仪表盘处理器。
func New(service *dashboardservice.Service, authService *authservice.AuthService) *Handler {
	return &Handler{service: service, authService: authService}
}

// GetSummary 查询仪表盘摘要。
func (h *Handler) GetSummary(ctx *gin.Context) {
	userID, ok := ctxmeta.CurrentUserID(ctx.Request.Context())
	if !ok {
		result.Error(ctx, apperr.ErrUnauthorized.HTTPStatus, apperr.ErrUnauthorized.Code, apperr.ErrUnauthorized.Message)
		return
	}
	userView, err := h.authService.GetCurrentUser(ctx.Request.Context(), userID)
	if err != nil {
		writeError(ctx, err)
		return
	}
	response, err := h.service.GetSummary(ctx.Request.Context(), &authentity.User{ID: userView.ID, Name: userView.Name, Role: userView.Role, ClassID: userView.ClassID, ClassName: userView.ClassName, Target: userView.Target})
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
