package config

// ObservabilityConfig 定义可观测性相关配置。
// 当前阶段优先支持访问日志增强、构建信息输出与调试入口开关。
type ObservabilityConfig struct {
	BuildVersion       string
	BuildCommit        string
	BuildTime          string
	EnablePprof        bool
	EnableMetrics      bool
	EnableTracing      bool
	SlowQueryThreshold string
}
