package graphwriter

import (
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
)

// Module 聚合图谱异步写入模块依赖。
type Module struct {
	Service *graphwriterservice.Service
}

// New 创建图谱异步写入模块。
func New(service *graphwriterservice.Service) *Module {
	return &Module{Service: service}
}
