package dto

// ExtractProfileRequest carries a natural-language learner description.
type ExtractProfileRequest struct {
	Message string `json:"message" binding:"required,max=2000"`
}

// UpsertProfileRequest persists confirmed profile dimensions.
type UpsertProfileRequest struct {
	Summary    string                 `json:"summary" binding:"omitempty,max=1000"`
	SourceText string                 `json:"sourceText" binding:"omitempty,max=2000"`
	Dimensions []ProfileDimensionView `json:"dimensions" binding:"required,min=1,max=20"`
}

// ProfileDimensionView describes one learner-profile dimension.
type ProfileDimensionView struct {
	Key        string  `json:"key" binding:"required,max=64"`
	Name       string  `json:"name" binding:"required,max=128"`
	Value      string  `json:"value" binding:"required,max=500"`
	Confidence float64 `json:"confidence" binding:"omitempty,min=0,max=1"`
	Evidence   string  `json:"evidence" binding:"omitempty,max=500"`
}

// LearningProfileView is the current structured learner profile.
type LearningProfileView struct {
	ID         int64                  `json:"id"`
	StudentID  int64                  `json:"studentId"`
	Summary    string                 `json:"summary"`
	SourceText string                 `json:"sourceText"`
	Dimensions []ProfileDimensionView `json:"dimensions"`
	UpdatedAt  string                 `json:"updatedAt"`
}

// ProfileResponse wraps the profile and completion status.
type ProfileResponse struct {
	Profile             *LearningProfileView `json:"profile"`
	HasCompletedProfile bool                 `json:"hasCompletedProfile"`
}
