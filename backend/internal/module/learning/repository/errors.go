package repository

import "errors"

var (
	// ErrDuplicateQuestionCode 表示题目标识在题库中已存在。
	ErrDuplicateQuestionCode = errors.New("duplicate question code")
	// ErrQuestionNotFound 表示题目不存在或已被禁用。
	ErrQuestionNotFound = errors.New("question not found")
	// ErrInvalidQuestion 表示答题请求中的题目、知识点或选项组合不合法。
	ErrInvalidQuestion = errors.New("invalid question")
	// ErrNotStudent 表示当前用户不是可参与学习流程的学生。
	ErrNotStudent = errors.New("not active student")
	// ErrForbiddenQuestion 表示当前用户没有访问该题目的权限。
	ErrForbiddenQuestion = errors.New("forbidden question")
)
