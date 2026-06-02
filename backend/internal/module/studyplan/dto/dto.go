package dto

import recommenddto "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/dto"

type GoalProfileRequest struct {
	GradeLevel         string   `json:"gradeLevel" binding:"required,max=32"`
	Major              string   `json:"major" binding:"omitempty,max=128"`
	GoalType           string   `json:"goalType" binding:"required,max=32"`
	GoalText           string   `json:"goalText" binding:"required,max=500"`
	TargetDirection    string   `json:"targetDirection" binding:"omitempty,max=255"`
	LearnedTopics      []string `json:"learnedTopics" binding:"omitempty,max=30"`
	FoundationLevel    string   `json:"foundationLevel" binding:"required,max=32"`
	WeeklyHours        int      `json:"weeklyHours" binding:"omitempty,min=1,max=80"`
	LearningPreference string   `json:"learningPreference" binding:"required,max=32"`
}

type GoalProfileView struct {
	ID                 int64    `json:"id"`
	GradeLevel         string   `json:"gradeLevel"`
	Major              string   `json:"major"`
	GoalType           string   `json:"goalType"`
	GoalText           string   `json:"goalText"`
	TargetDirection    string   `json:"targetDirection"`
	LearnedTopics      []string `json:"learnedTopics"`
	FoundationLevel    string   `json:"foundationLevel"`
	WeeklyHours        int      `json:"weeklyHours"`
	LearningPreference string   `json:"learningPreference"`
	UpdatedAt          string   `json:"updatedAt"`
}

type ProfileResponse struct {
	Profile             *GoalProfileView `json:"profile"`
	HasCompletedProfile bool             `json:"hasCompletedProfile"`
}

type StudyPlanStageView struct {
	Key     string                      `json:"key"`
	Title   string                      `json:"title"`
	Summary string                      `json:"summary"`
	Items   []recommenddto.PathItemView `json:"items"`
}

type StudyPlanView struct {
	ID          int64                       `json:"id"`
	Summary     string                      `json:"summary"`
	Stages      []StudyPlanStageView        `json:"stages"`
	NextSteps   []recommenddto.PathItemView `json:"nextSteps"`
	GeneratedAt string                      `json:"generatedAt"`
}

type CurrentPlanResponse struct {
	Profile *GoalProfileView `json:"profile,omitempty"`
	Plan    *StudyPlanView   `json:"plan,omitempty"`
}
