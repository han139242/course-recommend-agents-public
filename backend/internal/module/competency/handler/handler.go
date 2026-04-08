package handler

import (
	"github.com/gin-gonic/gin"
	authservice "github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供能力画像 HTTP 处理能力。
type Handler struct {
	service     *competencyservice.Service
	authService *authservice.AuthService
}

// New 创建能力画像处理器。
func New(service *competencyservice.Service, authService *authservice.AuthService) *Handler {
	return &Handler{service: service, authService: authService}
}

// GetProfile 查询当前学生能力画像。
func (h *Handler) GetProfile(ctx *gin.Context) {
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
	response, err := h.service.GetProfile(ctx.Request.Context(), &competencyentity.UserProfile{ID: userView.ID, Name: userView.Name, Role: userView.Role, ClassID: userView.ClassID, ClassName: userView.ClassName, Target: userView.Target})
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
