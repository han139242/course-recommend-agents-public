package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供认证模块的 HTTP 处理能力。
type Handler struct {
	authService *service.AuthService
}

// New 创建认证处理器。
func New(authService *service.AuthService) *Handler {
	return &Handler{authService: authService}
}

// Login 处理登录请求。
func (h *Handler) Login(ctx *gin.Context) {
	var request dto.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	response, err := h.authService.Login(request.Email, request.Password)
	if err != nil {
		h.writeError(ctx, err)
		return
	}

	result.Success(ctx, response)
}

// Me 返回当前登录用户信息。
func (h *Handler) Me(ctx *gin.Context) {
	userID, ok := ctxmeta.CurrentUserID(ctx.Request.Context())
	if !ok {
		result.Error(ctx, apperr.ErrUnauthorized.HTTPStatus, apperr.ErrUnauthorized.Code, apperr.ErrUnauthorized.Message)
		return
	}

	response, err := h.authService.GetCurrentUser(userID)
	if err != nil {
		h.writeError(ctx, err)
		return
	}

	result.Success(ctx, response)
}

func (h *Handler) writeError(ctx *gin.Context, err error) {
	appErr := apperr.AsAppError(err)
	result.Error(ctx, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
