package service

import (
	"context"
	"testing"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	"github.com/han139242/course-recommend-agents/backend/internal/module/auth/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

func TestRegisterCreatesStudentAndReturnsSession(t *testing.T) {
	passwords := NewPasswordService()
	seedHash, err := passwords.Hash("SeedPass123!")
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewMemoryUserRepository(seedHash, seedHash)
	jwt := NewJWTService(config.AuthConfig{JWTSecret: "test-secret-at-least-32-bytes-long", TokenTTL: time.Hour}, config.AppConfig{Name: "test"})
	service := NewAuthService(repo, jwt, passwords)

	response, err := service.Register(context.Background(), "王同学", " New.Student@Example.com ", "CoursePass123!")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if response.User.Role != "student" || response.User.Email != "new.student@example.com" || response.AccessToken == "" {
		t.Fatalf("unexpected registration response: %#v", response)
	}
	if _, err := service.Login(context.Background(), "new.student@example.com", "CoursePass123!"); err != nil {
		t.Fatalf("new account cannot login: %v", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	passwords := NewPasswordService()
	seedHash, _ := passwords.Hash("SeedPass123!")
	repo := repository.NewMemoryUserRepository(seedHash, seedHash)
	jwt := NewJWTService(config.AuthConfig{JWTSecret: "test-secret-at-least-32-bytes-long", TokenTTL: time.Hour}, config.AppConfig{Name: "test"})
	service := NewAuthService(repo, jwt, passwords)

	_, err := service.Register(context.Background(), "重复账号", "student@example.com", "CoursePass123!")
	if got := apperr.AsAppError(err); got.Code != apperr.CodeConflict {
		t.Fatalf("expected conflict, got %#v", err)
	}
}
