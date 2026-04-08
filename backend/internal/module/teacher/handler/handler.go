package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	teacherdto "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/dto"
	teacherservice "github.com/han139242/course-recommend-agents/backend/internal/module/teacher/service"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
	"github.com/han139242/course-recommend-agents/backend/pkg/result"
)

// Handler 提供教师端 HTTP 处理能力。
type Handler struct {
	service *teacherservice.TeacherService
}

// New 创建教师端处理器。
func New(service *teacherservice.TeacherService) *Handler {
	return &Handler{service: service}
}

// CreateClassroom 创建班级。
func (h *Handler) CreateClassroom(ctx *gin.Context) {
	var request teacherdto.CreateClassroomRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	response, err := h.service.CreateClassroom(ctx.Request.Context(), request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// ListClassrooms 查询班级列表。
func (h *Handler) ListClassrooms(ctx *gin.Context) {
	var query teacherdto.PaginationQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	response, err := h.service.ListClassrooms(ctx.Request.Context(), query)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// BatchCreateStudents 批量创建学生。
func (h *Handler) BatchCreateStudents(ctx *gin.Context) {
	classID, err := parseClassID(ctx.Param("classId"))
	if err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	var request teacherdto.BatchCreateStudentsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	response, err := h.service.BatchCreateStudents(ctx.Request.Context(), classID, request)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

// ListStudents 查询班级学生列表。
func (h *Handler) ListStudents(ctx *gin.Context) {
	classID, err := parseClassID(ctx.Param("classId"))
	if err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	var query teacherdto.PaginationQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		result.Error(ctx, apperr.ErrBadRequest.HTTPStatus, apperr.ErrBadRequest.Code, apperr.ErrBadRequest.Message)
		return
	}

	response, err := h.service.ListStudents(ctx.Request.Context(), classID, query)
	if err != nil {
		writeError(ctx, err)
		return
	}
	result.Success(ctx, response)
}

func parseClassID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func writeError(ctx *gin.Context, err error) {
	appErr := apperr.AsAppError(err)
	result.Error(ctx, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
