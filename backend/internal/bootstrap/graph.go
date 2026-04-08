package bootstrap

import (
	graphmodule "github.com/han139242/course-recommend-agents/backend/internal/module/graph"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NewGraphModule 初始化图谱模块。
func NewGraphModule(driver neo4j.DriverWithContext) *graphmodule.Module {
	return graphmodule.New(driver)
}
