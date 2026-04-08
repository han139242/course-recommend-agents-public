package bootstrap

import (
	learningmodule "github.com/han139242/course-recommend-agents/backend/internal/module/learning"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NewLearningModule 初始化学习模块。
// 学习模块允许接收可为空的 Redis 客户端，以支持缓存不可用时的降级运行。
func NewLearningModule(db *gorm.DB, redisClient *redis.Client) *learningmodule.Module {
	return learningmodule.New(db, redisClient)
}
