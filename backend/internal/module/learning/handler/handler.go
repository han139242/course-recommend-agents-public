package handler

import (
	"github.com/gin-gonic/gin"
	learningdto "github.com/han139242/course-recommend-agents/backend/internal/module/learning/dto"
	learningservice "github.com/han139242/course-recommend-agents/backend/internal/module/learning/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供学习模块 HTTP 处理能力。
type Handler struct {
	service *learningservice.Service
}

// New 创建学习模块处理器。
func New(service *learningservice.Service) *Handler {
	return &Handler{service: service}
}

// ImportQuestions 处理教师批量导题请求。
func (h *Handler) ImportQuestions(ctx *gin.Context) {
	var request learningdto.ImportQuestionsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.ImportQuestions(ctx.Request.Context(), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// ListQuestions 处理教师题目列表请求。
func (h *Handler) ListQuestions(ctx *gin.Context) {
	var query learningdto.ListQuestionsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.ListQuestions(ctx.Request.Context(), query)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// GetQuestionDetail 处理教师单题详情请求。
func (h *Handler) GetQuestionDetail(ctx *gin.Context) {
	response, err := h.service.GetQuestionDetail(ctx.Request.Context(), ctx.Param("questionId"))
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// ListStudentQuestions 处理学生拉题请求。
func (h *Handler) ListStudentQuestions(ctx *gin.Context) {
	var query learningdto.StudentQuestionsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.ListStudentQuestions(ctx.Request.Context(), query)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// SubmitAnswer 处理学生答题提交请求。
func (h *Handler) SubmitAnswer(ctx *gin.Context) {
	var request learningdto.SubmitAnswerRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}
	response, err := h.service.SubmitAnswer(ctx.Request.Context(), request)
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
