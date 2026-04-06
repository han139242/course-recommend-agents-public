package dto

// LoginResponse 定义登录响应。
type LoginResponse struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType"`
	ExpiresIn   int64        `json:"expiresIn"`
	User        AuthUserView `json:"user"`
}

// AuthUserView 定义当前用户视图。
type AuthUserView struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	Title     string  `json:"title"`
	StudentNo *string `json:"studentNo"`
	ClassID   *int64  `json:"classId"`
	ClassName *string `json:"className"`
	Target    string  `json:"target"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"createdAt"`
}
