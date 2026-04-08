package dto

// BatchCreateStudentsRequest 定义批量创建学生请求。
type BatchCreateStudentsRequest struct {
	Items []CreateStudentItem `json:"items" binding:"required,min=1,max=100"`
}

// CreateStudentItem 定义单个学生创建项。
type CreateStudentItem struct {
	Name      string `json:"name" binding:"required,max=64"`
	StudentNo string `json:"studentNo" binding:"required,max=64"`
	Email     string `json:"email" binding:"required,email,max=128"`
	Password  string `json:"password" binding:"required,min=6,max=72"`
	Target    string `json:"target" binding:"omitempty,max=255"`
}

// StudentView 定义学生视图。
type StudentView struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	StudentNo string `json:"studentNo"`
	Email     string `json:"email"`
	Target    string `json:"target"`
	CreatedAt string `json:"createdAt"`
}

// StudentDetailView 定义单个学生档案视图。
type StudentDetailView struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	Title     *string `json:"title"`
	StudentNo string  `json:"studentNo"`
	ClassID   int64   `json:"classId"`
	ClassName string  `json:"className"`
	Target    string  `json:"target"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"createdAt"`
}

// ListStudentsResponse 定义学生列表响应。
type ListStudentsResponse struct {
	List     []StudentView `json:"list"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

// BatchCreateStudentsResponse 定义批量创建学生响应。
type BatchCreateStudentsResponse struct {
	ClassID      int64 `json:"classId"`
	CreatedCount int   `json:"createdCount"`
}
