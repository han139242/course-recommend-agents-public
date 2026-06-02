package repository

import (
	"context"
	"math"
	"strconv"
	"time"

	learningentity "github.com/han139242/course-recommend-agents/backend/internal/module/learning/entity"
	"github.com/han139242/course-recommend-agents/backend/pkg/constant"
	"github.com/han139242/course-recommend-agents/backend/pkg/key"
	"github.com/redis/go-redis/v9"
)

const (
	masteryEvidenceTarget  = 8.0
	masteryConfidenceFloor = 0.35
	masteryPrecisionFactor = 100
)

// RedisRepository 提供学习状态相关的 Redis 操作。
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository 创建学习状态 Redis 仓储。
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// Enabled 返回当前是否可用 Redis 客户端。
func (r *RedisRepository) Enabled() bool {
	return r != nil && r.client != nil
}

// RecordAnswer 原子记录一次答题状态，并回写带题量置信度的 masteryScore。
func (r *RedisRepository) RecordAnswer(ctx context.Context, studentID int64, knowledgeID string, correct bool, answeredAt time.Time) (learningentity.MasteryState, error) {
	if !r.Enabled() {
		return learningentity.MasteryState{}, redis.Nil
	}
	correctInt := 0
	if correct {
		correctInt = 1
	}
	script := redis.NewScript(`
local answered = redis.call('HINCRBY', KEYS[1], ARGV[1], 1)
local correct = redis.call('HINCRBY', KEYS[2], ARGV[1], ARGV[2])
redis.call('HSET', KEYS[3], ARGV[1], ARGV[3])
local mastery = 0
if answered > 0 then
  local correctRate = tonumber(correct) / tonumber(answered)
  local evidence = tonumber(answered) / 8
  if evidence > 1 then
    evidence = 1
  end
  local confidence = 0.35 + 0.65 * evidence
  mastery = correctRate * 100 * confidence
  mastery = math.floor(mastery * 100 + 0.5) / 100
end
redis.call('HSET', KEYS[4], ARGV[1], tostring(mastery))
return {answered, correct, tostring(mastery)}
`)
	values, err := script.Run(ctx, r.client, []string{key.UserKnowledgeProgressAnswered(studentID), key.UserKnowledgeProgressCorrect(studentID), key.UserKnowledgeProgressLastAnswered(studentID), key.UserKnowledgeProgress(studentID)}, knowledgeID, correctInt, answeredAt.Format(time.RFC3339)).Result()
	if err != nil {
		return learningentity.MasteryState{}, err
	}
	items, ok := values.([]interface{})
	if !ok || len(items) != 3 {
		return learningentity.MasteryState{}, redis.ErrClosed
	}
	answeredCount, _ := toInt(items[0])
	correctCount, _ := toInt(items[1])
	masteryScore, _ := toFloat(items[2])
	return learningentity.MasteryState{KnowledgeID: knowledgeID, MasteryScore: masteryScore, CorrectCount: correctCount, AnsweredCount: answeredCount, LastAnsweredAt: &answeredAt}, nil
}

func evidenceAdjustedMastery(correctCount int, answeredCount int) float64 {
	if answeredCount <= 0 || correctCount <= 0 {
		return 0
	}
	correctRate := float64(correctCount) / float64(answeredCount)
	evidence := math.Min(1, float64(answeredCount)/masteryEvidenceTarget)
	confidence := masteryConfidenceFloor + (1-masteryConfidenceFloor)*evidence
	return math.Round(correctRate*100*confidence*masteryPrecisionFactor) / masteryPrecisionFactor
}

// GetMasteryVector 读取学生掌握度向量。
func (r *RedisRepository) GetMasteryVector(ctx context.Context, studentID int64) (map[string]float64, error) {
	if !r.Enabled() {
		return nil, redis.Nil
	}
	values, err := r.client.HGetAll(ctx, key.UserKnowledgeProgress(studentID)).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64, len(values))
	for field, raw := range values {
		parsed, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil {
			result[field] = constant.RedisHashFieldMasteryDefault
			continue
		}
		result[field] = parsed
	}
	return result, nil
}

// GetMasteryStates 读取学生知识点实时状态。
func (r *RedisRepository) GetMasteryStates(ctx context.Context, studentID int64) (map[string]learningentity.MasteryState, error) {
	if !r.Enabled() {
		return nil, redis.Nil
	}
	vector, err := r.GetMasteryVector(ctx, studentID)
	if err != nil {
		return nil, err
	}
	answeredMap, err := r.client.HGetAll(ctx, key.UserKnowledgeProgressAnswered(studentID)).Result()
	if err != nil {
		return nil, err
	}
	correctMap, err := r.client.HGetAll(ctx, key.UserKnowledgeProgressCorrect(studentID)).Result()
	if err != nil {
		return nil, err
	}
	lastMap, err := r.client.HGetAll(ctx, key.UserKnowledgeProgressLastAnswered(studentID)).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]learningentity.MasteryState, len(vector))
	for knowledgeID, score := range vector {
		answeredCount, _ := strconv.Atoi(answeredMap[knowledgeID])
		correctCount, _ := strconv.Atoi(correctMap[knowledgeID])
		state := learningentity.MasteryState{KnowledgeID: knowledgeID, MasteryScore: score, AnsweredCount: answeredCount, CorrectCount: correctCount}
		if raw := lastMap[knowledgeID]; raw != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, raw); parseErr == nil {
				state.LastAnsweredAt = &parsed
			}
		}
		result[knowledgeID] = state
	}
	return result, nil
}

// CacheCompetencyProfile 缓存能力画像 JSON。
func (r *RedisRepository) CacheCompetencyProfile(ctx context.Context, studentID int64, payload string) error {
	if !r.Enabled() {
		return redis.Nil
	}
	return r.client.Set(ctx, key.UserCompetencyProfile(studentID), payload, constant.RedisTTLCompetencyProfile).Err()
}

// GetCompetencyProfile 读取能力画像缓存。
func (r *RedisRepository) GetCompetencyProfile(ctx context.Context, studentID int64) (string, error) {
	if !r.Enabled() {
		return "", redis.Nil
	}
	return r.client.Get(ctx, key.UserCompetencyProfile(studentID)).Result()
}

// CacheRecommendPath 缓存推荐路径 JSON。
func (r *RedisRepository) CacheRecommendPath(ctx context.Context, studentID int64, targetID string, scene string, limit int, payload string) error {
	if !r.Enabled() {
		return redis.Nil
	}
	return r.client.Set(ctx, key.UserRecommendPath(studentID, targetID, scene, limit), payload, constant.RedisTTLRecommendPath).Err()
}

// GetRecommendPath 读取推荐路径缓存。
func (r *RedisRepository) GetRecommendPath(ctx context.Context, studentID int64, targetID string, scene string, limit int) (string, error) {
	if !r.Enabled() {
		return "", redis.Nil
	}
	return r.client.Get(ctx, key.UserRecommendPath(studentID, targetID, scene, limit)).Result()
}

func toInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, nil
	}
}

func toFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	case int64:
		return float64(v), nil
	default:
		return 0, nil
	}
}
