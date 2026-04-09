package service

import (
	"context"
	"testing"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	graphwriterentity "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/entity"
	graphwriterrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/repository"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

func TestEnqueueReturnsFalseWhenDisabled(t *testing.T) {
	service := New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: false})
	accepted := service.Enqueue(context.Background(), graphwriterentity.ProfileUpdateEvent{})
	if accepted {
		t.Fatal("expected enqueue to be rejected when service disabled")
	}
}

func TestEnqueueReturnsFalseBeforeStart(t *testing.T) {
	service := New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: true})
	accepted := service.Enqueue(context.Background(), graphwriterentity.ProfileUpdateEvent{})
	if accepted {
		t.Fatal("expected enqueue to be rejected before start")
	}
}

func TestApplyTopologyUpdateUsesJSONVersion(t *testing.T) {
	service := New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: true})
	service.applyTopologyUpdate(context.Background(), `{"version":"topology-v2"}`)
	snapshot := service.CurrentTopology()
	if snapshot.Version != "topology-v2" {
		t.Fatalf("unexpected topology version: %s", snapshot.Version)
	}
	if snapshot.UpdatedAt.IsZero() {
		t.Fatal("expected updated time to be set")
	}
}

func TestExtractTopologyVersionFallsBackToPayload(t *testing.T) {
	version := extractTopologyVersion("plain-version")
	if version != "plain-version" {
		t.Fatalf("unexpected version: %s", version)
	}
}

func TestCurrentTopologyHasBootstrapValue(t *testing.T) {
	service := New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: true, FlushInterval: time.Second})
	snapshot := service.CurrentTopology()
	if snapshot.Version == "" {
		t.Fatal("expected bootstrap topology version")
	}
}

func TestRuntimeStatusReflectsQueueAndLifecycle(t *testing.T) {
	service := New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: true, QueueSize: 4})
	before := service.RuntimeStatus()
	if before.Started {
		t.Fatal("expected service to be stopped before start")
	}
	if before.QueueCapacity != 4 {
		t.Fatalf("unexpected queue capacity before start: %d", before.QueueCapacity)
	}
}

func TestStopIsIdempotentWithoutStart(t *testing.T) {
	service := New(applogger.New("debug"), graphwriterrepo.NewNeo4jWriterRepository(nil), nil, config.GraphWriterConfig{Enabled: true})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Stop(ctx); err != nil {
		t.Fatalf("first stop failed: %v", err)
	}
	if err := service.Stop(ctx); err != nil {
		t.Fatalf("second stop failed: %v", err)
	}
}
