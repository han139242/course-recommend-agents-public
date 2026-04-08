package repository

import "errors"

var (
	ErrDuplicateQuestionCode = errors.New("duplicate question code")
	ErrQuestionNotFound      = errors.New("question not found")
	ErrInvalidQuestion       = errors.New("invalid question")
	ErrNotStudent            = errors.New("not active student")
	ErrForbiddenQuestion     = errors.New("forbidden question")
)
