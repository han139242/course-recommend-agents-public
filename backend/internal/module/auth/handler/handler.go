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
//
// 处理器只负责协议层职责，例如参数绑定、结果输出和错误映射，
// 业务决策全部下沉到服务层，避免 HTTP 细节反向污染领域逻辑。
type Handler struct {
	authService *service.AuthService
}

// New 创建认证处理器。
//
// 通过构造函数注入服务，便于在接口测试中替换实现并验证不同错误分支。
func New(authService *service.AuthService) *Handler {
	return &Handler{authService: authService}
}

// Login 处理登录请求。
//
// 登录接口在入口处只做基础结构校验，账号合法性、状态检查和令牌签发都交由服务层统一处理，
// 这样可以确保不同调用入口下的认证语义保持一致。
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
//
// 当前用户标识必须由鉴权中间件提前写入上下文；若缺失，说明请求未正确经过受保护链路，
// 此时直接返回未授权而不是继续执行业务查询，避免把中间件缺口吞掉。
func (h *Handler) Me(ctx *gin.Context) {
	userID, ok := ctxmeta.CurrentUserID(ctx.Request.Context())
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
