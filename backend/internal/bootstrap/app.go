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

// App bundles runtime dependencies for the HTTP service.
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

// New wires the application and prepares the HTTP server.
func New(ctx context.Context) (*App, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	logger := NewLogger(cfg)
	logStep := func(step string, fields ...applogger.Field) {
		logger.WithCtx(ctx).Info("bootstrap step", append([]applogger.Field{
			applogger.String("step", step),
		}, fields...)...)
	}

	logStep("config_loaded", applogger.String("http_addr", fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port)))

	logStep("mysql_connecting")
	mysqlDB, err := NewMySQL(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize MySQL: %w", err)
	}
	logStep("mysql_ready")

	logStep("auth_initializing")
	authModule, err := NewAuthModule(cfg, mysqlDB, logger)
	if err != nil {
		return nil, err
	}
	logStep("auth_ready")

	logStep("redis_connecting")
	redisClient, err := NewRedis(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize Redis: %w", err)
	}
	logStep("redis_ready", applogger.Bool("enabled", redisClient != nil))

	logStep("neo4j_connecting")
	neo4jDriver, err := NewNeo4j(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize Neo4j: %w", err)
	}
	logStep("neo4j_ready", applogger.Bool("enabled", neo4jDriver != nil))

	logStep("modules_initializing")
	teacherModule := NewTeacherModule(mysqlDB, redisClient)
	graphModule := NewGraphModule(neo4jDriver)
	graphWriterService := graphwriterservice.New(logger, graphwriterrepo.NewNeo4jWriterRepository(neo4jDriver), redisClient, cfg.GraphWriter)
	graphWriterModule := graphwritermodule.New(graphWriterService)
	competencyModule := NewCompetencyModule(logger, redisClient, authModule.AuthService, graphModule.Repo, graphWriterService, cfg.GraphWriter.TopologyChannel)
	learningModule := NewLearningModule(mysqlDB, redisClient, authModule.AuthService, graphWriterService, graphModule.Service)
	learningModule.SetCompetencyService(competencyModule.Service)
	go competencyModule.Matrix.Start(context.TODO())
	recommendModule := NewRecommendModule(neo4jDriver, redisClient, mysqlDB)
	studyPlanModule := NewStudyPlanModule(mysqlDB, recommendModule.Service)
	evaluationModule := NewEvaluationModule(recommendModule.Service.Scorer)
	dashboardModule := NewDashboardModule(competencyModule.Service, recommendModule.Service, authModule.AuthService)
	frontierModule := frontiermodule.New(graphModule.Repo)
	assistantModule := NewAssistantModule(cfg, authModule.AuthService, learningModule.Service, competencyModule.Service, recommendModule.Service, dashboardModule.Service, graphModule.Service, frontierModule.Service, mysqlDB)
	graphWriterService.Start(context.TODO())
	logStep("modules_ready")

	logStep("router_initializing")
	r := NewRouter(logger, cfg, mysqlDB, redisClient, neo4jDriver, authModule, teacherModule, learningModule, graphModule, graphWriterService, competencyModule, recommendModule, studyPlanModule, dashboardModule, frontierModule, assistantModule, evaluationModule)
	logStep("router_ready")

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}
	logStep("http_server_ready", applogger.String("addr", server.Addr))

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
