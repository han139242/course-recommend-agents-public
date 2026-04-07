package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/dto"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

const currentUserIDKey = "current_user_id"

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

	response, err := h.authService.Login(ctx.Request.Context(), request.Email, request.Password)
	if err != nil {
		h.writeError(ctx, err)
		return
	}

	result.Success(ctx, response)
}

// Me 返回当前登录用户信息。
func (h *Handler) Me(ctx *gin.Context) {
	userIDValue, exists := ctx.Get(currentUserIDKey)
	if !exists {
		result.Error(ctx, apperr.ErrUnauthorized.HTTPStatus, apperr.ErrUnauthorized.Code, apperr.ErrUnauthorized.Message)
		return
	}

	userID, ok := userIDValue.(int64)
	if !ok {
		result.Error(ctx, apperr.ErrUnauthorized.HTTPStatus, apperr.ErrUnauthorized.Code, apperr.ErrUnauthorized.Message)
		return
	}

	response, err := h.authService.GetCurrentUser(ctx.Request.Context(), userID)
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

// SetCurrentUserID 将当前用户 ID 写入 Gin 上下文。
func SetCurrentUserID(ctx *gin.Context, userID int64) {
	ctx.Set(currentUserIDKey, userID)
}

// ParseCurrentUserID 将字符串用户 ID 转成 int64。
func ParseCurrentUserID(raw string) (int64, error) {
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, errors.New("invalid user id")
	}
	return parsed, nil
}
