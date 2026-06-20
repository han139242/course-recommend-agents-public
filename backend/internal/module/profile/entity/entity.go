package entity

import "time"

// LearningProfile stores the latest confirmed structured learner profile.
type LearningProfile struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	StudentID  int64     `gorm:"column:student_id;not null;uniqueIndex:uk_student_learning_profiles_student"`
	Summary    string    `gorm:"column:summary;type:varchar(1000);not null;default:''"`
	SourceText string    `gorm:"column:source_text;type:text;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (LearningProfile) TableName() string {
	return "student_learning_profiles"
}

// ProfileDimension stores a normalized profile dimension and evidence.
type ProfileDimension struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ProfileID  int64     `gorm:"column:profile_id;not null;index:idx_profile_dimensions_profile_id"`
	StudentID  int64     `gorm:"column:student_id;not null;uniqueIndex:uk_profile_dimensions_student_key"`
	Key        string    `gorm:"column:dimension_key;type:varchar(64);not null;uniqueIndex:uk_profile_dimensions_student_key"`
	Name       string    `gorm:"column:dimension_name;type:varchar(128);not null"`
	Value      string    `gorm:"column:dimension_value;type:varchar(500);not null"`
	Confidence float64   `gorm:"column:confidence;not null;default:0"`
	Evidence   string    `gorm:"column:evidence;type:varchar(500);not null;default:''"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ProfileDimension) TableName() string {
	return "student_profile_dimensions"
}

// ProfileVersion stores lightweight profile-change history.
type ProfileVersion struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ProfileID   int64     `gorm:"column:profile_id;not null;index:idx_profile_versions_profile_id"`
	StudentID   int64     `gorm:"column:student_id;not null;index:idx_profile_versions_student_id"`
	Summary     string    `gorm:"column:summary;type:varchar(1000);not null;default:''"`
	PayloadJSON string    `gorm:"column:payload_json;type:text;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ProfileVersion) TableName() string {
	return "student_profile_versions"
}

// LearningEvent mirrors the shared learning_events table so the profile service
// can refresh dimensions from recent behavior without depending on resourcegen.
type LearningEvent struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	StudentID     int64     `gorm:"column:student_id;not null"`
	EventType     string    `gorm:"column:event_type;type:varchar(64);not null"`
	KnowledgeID   string    `gorm:"column:knowledge_id;type:varchar(128);not null;default:''"`
	KnowledgeName string    `gorm:"column:knowledge_name;type:varchar(255);not null;default:''"`
	ResourceID    int64     `gorm:"column:resource_id;not null;default:0"`
	ResourceType  string    `gorm:"column:resource_type;type:varchar(64);not null;default:''"`
	PayloadJSON   string    `gorm:"column:payload_json;type:text;not null"`
	OccurredAt    time.Time `gorm:"column:occurred_at;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (LearningEvent) TableName() string {
	return "learning_events"
}
