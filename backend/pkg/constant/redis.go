package constant

import "time"

const (
	// RedisHashFieldMasteryDefault 表示掌握度默认值。
	RedisHashFieldMasteryDefault = 0.0
	// RedisCompetencyScale 表示能力分值的展示比例尺。
	RedisCompetencyScale = 100.0
)

const (
	// RedisTTLCompetencyProfile 表示能力画像缓存 TTL。
	RedisTTLCompetencyProfile = 10 * time.Minute
	// RedisTTLRecommendPath 表示推荐路径缓存 TTL。
	RedisTTLRecommendPath = 5 * time.Minute
)
