package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	graphwriterentity "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/entity"
	graphwriterrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/repository"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// TopologySnapshot 表示当前已加载的拓扑版本快照。
type TopologySnapshot struct {
	Version   string
	UpdatedAt time.Time
	Payload   string
}

// Service 提供用户能力图谱异步写入与拓扑热更新能力。
type Service struct {
	logger   *applogger.Logger
	repo     *graphwriterrepo.Neo4jWriterRepository
	redis    *redis.Client
	config   config.GraphWriterConfig

	queue        chan graphwriterentity.ProfileUpdateEvent
	stopCh       chan struct{}
	workerCtx    context.Context
	workerCancel context.CancelFunc
	stopDone     chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup

	started          atomic.Bool
	stopped          atomic.Bool
	subscriberActive atomic.Bool
	workerActive     atomic.Bool
	lastFlushError   atomic.Value
	lastFlushAt      atomic.Int64
	lastErrorAt      atomic.Int64

	topologyValue atomic.Value
}

// New 创建图谱异步写入服务。
func New(logger *applogger.Logger, repo *graphwriterrepo.Neo4jWriterRepository, redisClient *redis.Client, cfg config.GraphWriterConfig) *Service {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 128
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 16
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 2 * time.Second
	}
	if cfg.RetryTimes <= 0 {
		cfg.RetryTimes = 3
	}
	if strings.TrimSpace(cfg.TopologyChannel) == "" {
		cfg.TopologyChannel = "topology_updates"
	}
	workerCtx, workerCancel := context.WithCancel(context.Background())
	service := &Service{
		logger:       logger,
		repo:         repo,
		redis:        redisClient,
		config:       cfg,
		queue:        make(chan graphwriterentity.ProfileUpdateEvent, cfg.QueueSize),
		stopCh:       make(chan struct{}),
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
		stopDone:     make(chan struct{}),
	}
	service.topologyValue.Store(TopologySnapshot{Version: "bootstrap", UpdatedAt: time.Now().UTC(), Payload: ""})
	service.lastFlushError.Store("")
	return service
}

// Enabled 返回当前服务是否启用。
func (s *Service) Enabled() bool {
	return s != nil && s.config.Enabled && s.repo != nil && s.repo.Enabled()
}

// Start 启动后台 worker 与拓扑订阅。
func (s *Service) Start(_ context.Context) {
	if !s.Enabled() || !s.started.CompareAndSwap(false, true) {
		return
	}
	if s.logger != nil {
		s.logger.WithCtx(s.workerCtx).Info("图谱异步写入器启动", applogger.Int("queue_size", cap(s.queue)), applogger.Int("batch_size", s.config.BatchSize))
	}
	if s.redis != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runSubscriber(s.workerCtx)
		}()
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runWorker(s.workerCtx)
	}()
	go func() {
		s.wg.Wait()
		close(s.stopDone)
	}()
}

// Stop 停止后台 worker，并在退出前尽量刷完缓冲任务。
func (s *Service) Stop(ctx context.Context) error {
	if s == nil || !s.Enabled() {
		return nil
	}
	s.stopOnce.Do(func() {
		s.stopped.Store(true)
		close(s.stopCh)
		if s.workerCancel != nil {
			s.workerCancel()
		}
	})
	if !s.started.Load() {
		return nil
	}
	select {
	case <-s.stopDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Enqueue 将一次画像更新加入异步队列。
func (s *Service) Enqueue(ctx context.Context, event graphwriterentity.ProfileUpdateEvent) bool {
	if !s.Enabled() || !s.started.Load() || s.stopped.Load() {
		return false
	}
	event.TopologyVersion = s.CurrentTopology().Version
	select {
	case <-s.stopCh:
		return false
	case s.queue <- event:
		return true
	default:
		if s.logger != nil {
			s.logger.WithCtx(ctx).Warn("图谱异步写入队列已满，丢弃本次更新", applogger.Int64("student_id", event.StudentID), applogger.Int("queue_size", len(s.queue)))
		}
		return false
	}
}

// CurrentTopology 返回当前拓扑快照。
func (s *Service) CurrentTopology() TopologySnapshot {
	if s == nil {
		return TopologySnapshot{}
	}
	value, ok := s.topologyValue.Load().(TopologySnapshot)
	if !ok {
		return TopologySnapshot{}
	}
	return value
}

// QueueLength 返回当前队列长度。
func (s *Service) QueueLength() int {
	if s == nil {
		return 0
	}
	return len(s.queue)
}

func (s *Service) runWorker(ctx context.Context) {
	s.workerActive.Store(true)
	defer s.workerActive.Store(false)
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()
	batch := make([]graphwriterentity.ProfileUpdateEvent, 0, s.config.BatchSize)
	flush := func(reason string) {
		if len(batch) == 0 {
			return
		}
		if err := s.flushBatch(ctx, batch, reason); err != nil && s.logger != nil {
			s.logger.WithCtx(ctx).Error("图谱批量写入失败", applogger.String("reason", reason), applogger.ErrorField("error", err), applogger.Int("batch_size", len(batch)))
		}
		batch = batch[:0]
	}
	for {
		select {
		case <-s.stopCh:
			drain := true
			for drain {
				select {
				case event, ok := <-s.queue:
					if !ok {
						drain = false
						continue
					}
					batch = append(batch, event)
					if len(batch) >= s.config.BatchSize {
						flush("shutdown_batch")
					}
				default:
					drain = false
				}
			}
			flush("shutdown_flush")
			return
		case event, ok := <-s.queue:
			if !ok {
				flush("queue_closed")
				return
			}
			batch = append(batch, event)
			if len(batch) >= s.config.BatchSize {
				flush("batch_full")
			}
		case <-ticker.C:
			flush("interval")
		}
	}
}

func (s *Service) flushBatch(ctx context.Context, batch []graphwriterentity.ProfileUpdateEvent, reason string) error {
	var lastErr error
	topologyVersion := s.CurrentTopology().Version
	for attempt := 1; attempt <= s.config.RetryTimes; attempt++ {
		err := s.repo.UpsertUserCompetencies(ctx, batch, topologyVersion)
		if err == nil {
			s.lastFlushAt.Store(time.Now().Unix())
			s.lastFlushError.Store("")
			if s.logger != nil {
				s.logger.WithCtx(ctx).Info("图谱批量写入完成", applogger.String("reason", reason), applogger.Int("batch_size", len(batch)), applogger.String("topology_version", topologyVersion), applogger.Int("queue_length", len(s.queue)))
			}
			return nil
		}
		lastErr = err
		s.lastFlushError.Store(err.Error())
		s.lastErrorAt.Store(time.Now().Unix())
		if s.logger != nil {
			s.logger.WithCtx(ctx).Warn("图谱批量写入重试", applogger.Int("attempt", attempt), applogger.ErrorField("error", err), applogger.Int("batch_size", len(batch)))
		}
		time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
	}
	if s.logger != nil {
		payload, _ := json.Marshal(batch)
		s.logger.WithCtx(ctx).Error("图谱批量写入进入死信日志", applogger.ErrorField("error", lastErr), applogger.Any("events", string(payload)))
	}
	return fmt.Errorf("图谱批量写入最终失败: %w", lastErr)
}

func (s *Service) runSubscriber(ctx context.Context) {
	s.subscriberActive.Store(true)
	defer s.subscriberActive.Store(false)
	pubsub := s.redis.Subscribe(ctx, s.config.TopologyChannel)
	defer pubsub.Close()
	channel := pubsub.Channel()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case message, ok := <-channel:
			if !ok {
				return
			}
			s.applyTopologyUpdate(ctx, message.Payload)
		}
	}
}

func (s *Service) applyTopologyUpdate(ctx context.Context, payload string) {
	version := extractTopologyVersion(payload)
	s.topologyValue.Store(TopologySnapshot{Version: version, UpdatedAt: time.Now().UTC(), Payload: payload})
	if s.logger != nil {
		s.logger.WithCtx(ctx).Info("已加载拓扑热更新", applogger.String("version", version), applogger.String("channel", s.config.TopologyChannel))
	}
}

func extractTopologyVersion(payload string) string {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return fmt.Sprintf("empty-%d", time.Now().Unix())
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(trimmed), &body); err == nil && strings.TrimSpace(body.Version) != "" {
		return strings.TrimSpace(body.Version)
	}
	return trimmed
}

// ErrNotReady 表示服务当前未启用。
var ErrNotReady = errors.New("图谱异步写入器未启用")
