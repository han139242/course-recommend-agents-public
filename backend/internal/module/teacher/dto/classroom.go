package dto

// CreateClassroomRequest 定义创建班级请求。
type CreateClassroomRequest struct {
	Name  string `json:"name" binding:"required,max=128"`
	Grade string `json:"grade" binding:"omitempty,max=64"`
}

// ClassroomView 定义班级视图。
type ClassroomView struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Grade        string `json:"grade"`
	StudentCount int64  `json:"studentCount"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}

// ListClassroomsResponse 定义班级列表响应。
type ListClassroomsResponse struct {
	List     []ClassroomView `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// CreateClassroomResponse 定义创建班级响应。
type CreateClassroomResponse struct {
	Classroom ClassroomView `json:"classroom"`
}
