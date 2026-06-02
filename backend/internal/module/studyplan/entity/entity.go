package entity

import "time"

// GoalProfile stores the student's current goal-planning profile.
type GoalProfile struct {
	ID                 int64     `gorm:"column:id;primaryKey;autoIncrement"`
	StudentID          int64     `gorm:"column:student_id;not null;uniqueIndex:uk_student_goal_profiles_student"`
	GradeLevel         string    `gorm:"column:grade_level;type:varchar(32);not null"`
	Major              string    `gorm:"column:major;type:varchar(128);not null;default:''"`
	GoalType           string    `gorm:"column:goal_type;type:varchar(32);not null"`
	GoalText           string    `gorm:"column:goal_text;type:varchar(500);not null"`
	TargetDirection    string    `gorm:"column:target_direction;type:varchar(255);not null;default:''"`
	LearnedTopicsJSON  string    `gorm:"column:learned_topics_json;type:text;not null"`
	FoundationLevel    string    `gorm:"column:foundation_level;type:varchar(32);not null"`
	WeeklyHours        int       `gorm:"column:weekly_hours;not null;default:6"`
	LearningPreference string    `gorm:"column:learning_preference;type:varchar(32);not null"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (GoalProfile) TableName() string {
	return "student_goal_profiles"
}

// StudyPlan stores the latest generated account-level study plan.
type StudyPlan struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	StudentID     int64     `gorm:"column:student_id;not null;uniqueIndex:uk_student_study_plans_student"`
	ProfileID     int64     `gorm:"column:profile_id;not null"`
	Summary       string    `gorm:"column:summary;type:varchar(500);not null"`
	StagesJSON    string    `gorm:"column:stages_json;type:text;not null"`
	NextStepsJSON string    `gorm:"column:next_steps_json;type:text;not null"`
	GeneratedAt   time.Time `gorm:"column:generated_at;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (StudyPlan) TableName() string {
	return "student_study_plans"
}
