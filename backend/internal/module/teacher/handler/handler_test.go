package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

func TestGetStudentDetailBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}
	r := gin.New()
	r.GET("/students/:studentId", h.GetStudentDetail)
	req := httptest.NewRequest(http.MethodGet, "/students/not-a-number", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBatchCreateStudentsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}
	r := gin.New()
	r.POST("/classes/:classId/students", h.BatchCreateStudents)
	req := httptest.NewRequest(http.MethodPost, "/classes/not-a-number/students", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestWriteErrorUsesAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request = req
	writeError(ctx, apperr.ErrConflict)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}
