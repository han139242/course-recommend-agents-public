package repository

import (
	"testing"
)

func TestEnabledWithNilClient(t *testing.T) {
	repo := NewRedisRepository(nil)
	if repo.Enabled() {
		t.Fatal("expected disabled redis repository")
	}
}
