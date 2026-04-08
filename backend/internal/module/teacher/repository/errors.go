package repository

import "errors"

var (
	// ErrClassroomNotFound 表示班级不存在。
	ErrClassroomNotFound = errors.New("classroom not found")
	// ErrStudentNotFound 表示学生不存在或不在当前教师作用域内。
	ErrStudentNotFound = errors.New("student not found")
	// ErrDuplicateEmail 表示邮箱已存在。
	ErrDuplicateEmail = errors.New("duplicate email")
	// ErrDuplicateStudentNo 表示学号已存在。
	ErrDuplicateStudentNo = errors.New("duplicate student no")
	// ErrDuplicateClassroomName 表示同一教师下班级名重复。
	ErrDuplicateClassroomName = errors.New("duplicate classroom name")
)
