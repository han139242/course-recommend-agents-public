package entity

import "time"

const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"

	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"

	ResourceStatusReady = "ready"
)

// GenerationJob stores one resource generation task.
type GenerationJob struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	StudentID     int64     `gorm:"column:student_id;not null;index:idx_resource_jobs_student_id"`
	KnowledgeID   string    `gorm:"column:knowledge_id;type:varchar(128);not null;default:'';index:idx_resource_jobs_knowledge_id"`
	KnowledgeName string    `gorm:"column:knowledge_name;type:varchar(255);not null;default:''"`
	CourseTitle   string    `gorm:"column:course_title;type:varchar(255);not null;default:''"`
	Target        string    `gorm:"column:target;type:varchar(500);not null;default:''"`
	Status        string    `gorm:"column:status;type:varchar(32);not null;default:'pending';index:idx_resource_jobs_status"`
	Progress      int       `gorm:"column:progress;not null;default:0"`
	CurrentAgent  string    `gorm:"column:current_agent;type:varchar(128);not null;default:''"`
	ErrorMessage  string    `gorm:"column:error_message;type:varchar(1000);not null;default:''"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (GenerationJob) TableName() string {
	return "resource_generation_jobs"
}

// AgentStep stores one orchestrated agent step.
type AgentStep struct {
	ID        int64      `gorm:"column:id;primaryKey;autoIncrement"`
	JobID     int64      `gorm:"column:job_id;not null;index:idx_resource_agent_steps_job_id"`
	StudentID int64      `gorm:"column:student_id;not null;index:idx_resource_agent_steps_student_id"`
	AgentKey  string     `gorm:"column:agent_key;type:varchar(64);not null"`
	AgentName string     `gorm:"column:agent_name;type:varchar(128);not null"`
	Role      string     `gorm:"column:role;type:varchar(255);not null;default:''"`
	Status    string     `gorm:"column:status;type:varchar(32);not null;default:'completed'"`
	Summary   string     `gorm:"column:summary;type:varchar(1000);not null;default:''"`
	StartedAt time.Time  `gorm:"column:started_at;not null"`
	EndedAt   *time.Time `gorm:"column:ended_at"`
}

func (AgentStep) TableName() string {
	return "resource_agent_steps"
}

// GeneratedResource stores one generated resource artifact.
type GeneratedResource struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	JobID         int64     `gorm:"column:job_id;not null;index:idx_generated_resources_job_id"`
	StudentID     int64     `gorm:"column:student_id;not null;index:idx_generated_resources_student_id"`
	ResourceType  string    `gorm:"column:resource_type;type:varchar(64);not null;index:idx_generated_resources_type"`
	TypeLabel     string    `gorm:"column:type_label;type:varchar(128);not null"`
	Title         string    `gorm:"column:title;type:varchar(255);not null"`
	Content       string    `gorm:"column:content;type:longtext;not null"`
	Format        string    `gorm:"column:format;type:varchar(32);not null;default:'markdown'"`
	KnowledgeID   string    `gorm:"column:knowledge_id;type:varchar(128);not null;default:'';index:idx_generated_resources_knowledge_id"`
	KnowledgeName string    `gorm:"column:knowledge_name;type:varchar(255);not null;default:''"`
	CourseTitle   string    `gorm:"column:course_title;type:varchar(255);not null;default:''"`
	Reason        string    `gorm:"column:reason;type:varchar(500);not null;default:''"`
	Status        string    `gorm:"column:status;type:varchar(32);not null;default:'ready'"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (GeneratedResource) TableName() string {
	return "generated_resources"
}

// ResourceFeedback stores learner feedback and learning-effect signals.
type ResourceFeedback struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ResourceID int64     `gorm:"column:resource_id;not null;uniqueIndex:uk_resource_feedback_student_resource"`
	JobID      int64     `gorm:"column:job_id;not null;index:idx_resource_feedback_job_id"`
	StudentID  int64     `gorm:"column:student_id;not null;uniqueIndex:uk_resource_feedback_student_resource;index:idx_resource_feedback_student_id"`
	Rating     int       `gorm:"column:rating;not null;default:0"`
	Difficulty string    `gorm:"column:difficulty;type:varchar(32);not null;default:''"`
	Helpful    bool      `gorm:"column:helpful;not null;default:false"`
	Completed  bool      `gorm:"column:completed;not null;default:false"`
	Comment    string    `gorm:"column:comment;type:varchar(500);not null;default:''"`
	NextAction string    `gorm:"column:next_action;type:varchar(500);not null;default:''"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ResourceFeedback) TableName() string {
	return "resource_feedbacks"
}
