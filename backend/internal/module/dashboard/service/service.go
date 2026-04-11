package service

import (
	"context"

	authentity "github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
	competencyservice "github.com/han139242/course-recommend-agents/backend/internal/module/competency/service"
	dashboarddto "github.com/han139242/course-recommend-agents/backend/internal/module/dashboard/dto"
	recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
)

// Service 提供首页仪表盘聚合能力。
type Service struct {
	competencyService *competencyservice.Service
	recommendService  *recommendservice.Service
}

// New 创建仪表盘服务。
func New(competencyService *competencyservice.Service, recommendService *recommendservice.Service) *Service {
	return &Service{competencyService: competencyService, recommendService: recommendService}
}

// GetSummary 查询摘要聚合视图。
func (s *Service) GetSummary(ctx context.Context, user *authentity.User) (*dashboarddto.SummaryResponse, error) {
	profileUser := competencyservice.UserToProfile(*user)
	profile, _ := s.competencyService.GetProfile(ctx, &profileUser)
	path, _ := s.recommendService.GetPath(ctx, recommenddto.PathQuery{Scene: "dashboard", Limit: 3, Target: user.Target})
	stats := profile.Profile.Stats
	cards := make([]dashboarddto.KnowledgeCardView, 0, len(path.Path))
	for _, item := range path.Path {
		cards = append(cards, dashboarddto.KnowledgeCardView{ID: item.Title, Type: item.Stage, Progress: item.Score * 100, Title: item.Title, Domain: "综合能力", Tag: item.ReasonLabel, Route: item.Route})
	}
	return &dashboarddto.SummaryResponse{QuickStats: stats, Stats: stats, KnowledgeCards: cards, RecommendedPath: path.Path}, nil
}
