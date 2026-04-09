package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/han139242/course-recommend-agents/backend/internal/config"
	frontiermodule "github.com/han139242/course-recommend-agents/backend/internal/module/frontier"
	graphmodule "github.com/han139242/course-recommend-agents/backend/internal/module/graph"
	graphwritermodule "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter"
	graphwriterrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/repository"
	graphwriterservice "github.com/han139242/course-recommend-agents/backend/internal/module/graphwriter/service"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// App 聚合应用运行期依赖。
//
// 该结构体只负责承载启动阶段已经准备好的基础设施对象，
// 便于入口层统一管理生命周期与停机顺序。
type App struct {
	Config       config.Config
	Logger       *applogger.Logger
	MySQL        *gorm.DB
	Redis        *redis.Client
	Neo4j        neo4j.DriverWithContext
	GraphWriter  *graphwritermodule.Module
	Frontier     *frontiermodule.Module
	Graph        *graphmodule.Module
	Router       http.Handler
	HTTPServer   *http.Server
	shutdownJobs []func(context.Context) error
}

// New 创建应用实例。
// 装配顺序遵循“先基础设施、再模块、最后路由与 HTTP 服务”的约定，
// 这样可以让模块只依赖已准备好的共享资源，避免在路由层直接拼装底层依赖。
func New(ctx context.Context) (*App, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	logger := NewLogger(cfg)

	mysqlDB, err := NewMySQL(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化 MySQL 失败: %w", err)
	}

	authModule, err := NewAuthModule(cfg, mysqlDB, logger)
	if err != nil {
		return nil, err
	}

	redisClient, err := NewRedis(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化 Redis 失败: %w", err)
	}

	neo4jDriver, err := NewNeo4j(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化 Neo4j 失败: %w", err)
	}

	teacherModule := NewTeacherModule(mysqlDB, redisClient)
	graphModule := NewGraphModule(neo4jDriver)
	graphWriterService := graphwriterservice.New(logger, graphwriterrepo.NewNeo4jWriterRepository(neo4jDriver), redisClient, cfg.GraphWriter)
	graphWriterModule := graphwritermodule.New(graphWriterService)
	competencyModule := NewCompetencyModule(logger, redisClient, authModule.AuthService, graphModule.Repo, graphWriterService, cfg.GraphWriter.TopologyChannel)
	learningModule := NewLearningModule(mysqlDB, redisClient, authModule.AuthService, graphWriterService)
	learningModule.SetCompetencyService(competencyModule.Service)
	competencyModule.Matrix.Start(context.TODO())
	recommendModule := NewRecommendModule(neo4jDriver, redisClient)
	dashboardModule := NewDashboardModule(competencyModule.Service, recommendModule.Service, authModule.AuthService)
	frontierModule := frontiermodule.New(graphModule.Repo)
	graphWriterService.Start(context.TODO())

	r := NewRouter(logger, cfg, mysqlDB, redisClient, neo4jDriver, authModule, teacherModule, learningModule, graphModule, graphWriterService, competencyModule, recommendModule, dashboardModule, frontierModule)
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	app := &App{
		Config:      cfg,
		Logger:      logger,
		MySQL:       mysqlDB,
		Redis:       redisClient,
		Neo4j:       neo4jDriver,
		GraphWriter: graphWriterModule,
		Frontier:    frontierModule,
		Graph:       graphModule,
		Router:      r,
		HTTPServer:  server,
	}

	app.registerShutdownJobs()
	return app, nil
}
