package service

import (
	"context"
	"testing"

	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	learningrepo "github.com/han139242/course-recommend-agents/backend/internal/module/learning/repository"
)

func TestGetProfileWithDisabledRedisStillReturnsProfile(t *testing.T) {
	service := New(learningrepo.NewRedisRepository(nil))
	user := &competencyentity.UserProfile{ID: 1, Name: "学生A", Role: "student", Target: "算法工程师"}
	resp, err := service.GetProfile(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Profile.Name != "学生A" {
		t.Fatalf("unexpected profile: %+v", resp.Profile)
	}
}
