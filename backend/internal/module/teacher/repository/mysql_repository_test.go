package repository

import "testing"

func TestTranslateWriteErrorFallbackKeepsOriginalError(t *testing.T) {
	err := translateWriteError(assertError("other"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
}

type plainError string

func (e plainError) Error() string { return string(e) }

func assertError(message string) error {
	return plainError(message)
}
