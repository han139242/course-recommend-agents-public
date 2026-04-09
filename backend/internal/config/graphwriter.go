package config

import "time"

// GraphWriterConfig 定义用户能力图谱异步写入配置。
// 这些参数共同决定内存队列容量、批量刷新节奏与失败重试策略。
type GraphWriterConfig struct {
	Enabled         bool
	QueueSize       int
	BatchSize       int
	FlushInterval   time.Duration
	RetryTimes      int
	TopologyChannel string
}
