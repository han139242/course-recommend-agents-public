package service

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// MatrixManager maintains the in-memory competency mapping snapshot.
type MatrixManager struct {
	logger          *applogger.Logger
	graphRepo       *graphrepo.Neo4jRepository
	redisClient     *redis.Client
	topologyChannel string
	active          atomic.Value
	shadow          atomic.Value
	ready           atomic.Bool
	subscriberOn    atomic.Bool
}

// NewMatrixManager creates the matrix manager.
func NewMatrixManager(logger *applogger.Logger, graphRepo *graphrepo.Neo4jRepository, redisClient *redis.Client, topologyChannel string) *MatrixManager {
	manager := &MatrixManager{logger: logger, graphRepo: graphRepo, redisClient: redisClient, topologyChannel: topologyChannel}
	empty := competencyentity.MatrixSnapshot{Version: "bootstrap", UpdatedAt: time.Now().UTC(), Mappings: map[string][]competencyentity.CompetencyWeight{}}
	manager.active.Store(empty)
	manager.shadow.Store(empty)
	return manager
}

// Current returns the active competency snapshot.
func (m *MatrixManager) Current() competencyentity.MatrixSnapshot {
	if m == nil {
		return competencyentity.MatrixSnapshot{}
	}
	if value, ok := m.active.Load().(competencyentity.MatrixSnapshot); ok {
		return value
	}
	return competencyentity.MatrixSnapshot{}
}

// Ready reports whether the matrix has loaded successfully.
func (m *MatrixManager) Ready() bool {
	return m != nil && m.ready.Load()
}

// SubscriberActive reports whether the topology subscriber is active.
func (m *MatrixManager) SubscriberActive() bool {
	return m != nil && m.subscriberOn.Load()
}

// Start loads the initial snapshot and subscribes to later updates.
func (m *MatrixManager) Start(ctx context.Context) {
	if m == nil {
		return
	}
	m.reload(ctx, "bootstrap")
	if m.redisClient == nil || m.topologyChannel == "" {
		return
	}
	go m.subscribe(ctx)
}

// ComputeScores aggregates competency scores from knowledge mastery.
func (m *MatrixManager) ComputeScores(mastery map[string]float64) []competencyentity.CompetencyScore {
	snapshot := m.Current()
	aggregated := make(map[string]competencyentity.CompetencyScore)
	for knowledgeID, score := range mastery {
		weights := snapshot.Mappings[knowledgeID]
		for _, weight := range weights {
			item := aggregated[weight.CompetencyID]
			item.CompetencyID = weight.CompetencyID
			item.CompetencyName = weight.CompetencyName
			item.Category = weight.Category
			item.Score += (score / 100.0) * weight.Weight
			aggregated[weight.CompetencyID] = item
		}
	}
	result := make([]competencyentity.CompetencyScore, 0, len(aggregated))
	for _, item := range aggregated {
		if item.Score > 1 {
			item.Score = 1
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].CompetencyID < result[j].CompetencyID
		}
		return result[i].Score > result[j].Score
	})
	return result
}

func (m *MatrixManager) subscribe(ctx context.Context) {
	m.subscriberOn.Store(true)
	defer m.subscriberOn.Store(false)
	pubsub := m.redisClient.Subscribe(ctx, m.topologyChannel)
	defer pubsub.Close()
	for message := range pubsub.Channel() {
		m.reload(ctx, message.Payload)
	}
}

func (m *MatrixManager) reload(ctx context.Context, version string) {
	if m == nil || m.graphRepo == nil || !m.graphRepo.Enabled() {
		return
	}

	reloadCtx := ctx
	cancel := func() {}
	if reloadCtx == nil {
		reloadCtx = context.Background()
	}
	if _, hasDeadline := reloadCtx.Deadline(); !hasDeadline {
		reloadCtx, cancel = context.WithTimeout(reloadCtx, 3*time.Second)
	}
	defer cancel()

	mappings, err := m.graphRepo.LoadCompetencyMappings(reloadCtx)
	if err != nil {
		if m.logger != nil {
			m.logger.WithCtx(reloadCtx).Warn("competency matrix refresh failed", applogger.String("version", version), applogger.ErrorField("error", err))
		}
		return
	}
	converted := make(map[string][]competencyentity.CompetencyWeight, len(mappings))
	for knowledgeID, items := range mappings {
		weights := make([]competencyentity.CompetencyWeight, 0, len(items))
		for _, item := range items {
			weights = append(weights, competencyentity.CompetencyWeight{
				CompetencyID:   item.CompetencyID,
				CompetencyName: item.CompetencyName,
				Category:       item.Category,
				Weight:         item.Weight,
			})
		}
		converted[knowledgeID] = weights
	}
	snapshot := competencyentity.MatrixSnapshot{Version: version, UpdatedAt: time.Now().UTC(), Mappings: converted}
	m.shadow.Store(snapshot)
	m.active.Store(snapshot)
	m.ready.Store(true)
}
