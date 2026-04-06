package apperr

import "net/http"

const (
	CodeBadRequest            = 10001
	CodeUnauthorized          = 10002
	CodeForbidden             = 10003
	CodeNotFound              = 10004
	CodeConflict              = 10005
	CodeDependencyUnavailable = 10006
	CodeTooManyRequests       = 10007
	CodeInternal              = 10008
)

// Error 定义应用统一错误模型。
type Error struct {
	HTTPStatus int
	Code       int
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	return e.Message
}

// WithCause 为应用错误补充底层原因。
func (e *Error) WithCause(err error) *Error {
	copied := *e
	copied.Cause = err
	return &copied
}

var (
	ErrBadRequest            = &Error{HTTPStatus: http.StatusBadRequest, Code: CodeBadRequest, Message: "参数错误"}
	ErrUnauthorized          = &Error{HTTPStatus: http.StatusUnauthorized, Code: CodeUnauthorized, Message: "未认证"}
	ErrForbidden             = &Error{HTTPStatus: http.StatusForbidden, Code: CodeForbidden, Message: "无权限"}
	ErrNotFound              = &Error{HTTPStatus: http.StatusNotFound, Code: CodeNotFound, Message: "资源不存在"}
	ErrConflict              = &Error{HTTPStatus: http.StatusConflict, Code: CodeConflict, Message: "资源冲突"}
	ErrDependencyUnavailable = &Error{HTTPStatus: http.StatusServiceUnavailable, Code: CodeDependencyUnavailable, Message: "依赖不可用"}
	ErrTooManyRequests       = &Error{HTTPStatus: http.StatusTooManyRequests, Code: CodeTooManyRequests, Message: "请求过于频繁"}
	ErrInternal              = &Error{HTTPStatus: http.StatusInternalServerError, Code: CodeInternal, Message: "系统繁忙，请稍后再试"}
)

// AsAppError 将任意错误转换为应用错误。
func AsAppError(err error) *Error {
	if err == nil {
		return nil
	}

	appErr, ok := err.(*Error)
	if ok {
		return appErr
	}

	return ErrInternal.WithCause(err)
}
