package dto

// LoginResponse 定义登录响应。
//
// 登录成功后同时返回令牌元信息与当前用户概要，目的是减少前端首次进入系统时的额外自举请求。
type LoginResponse struct {
	AccessToken string       `json:"accessToken"`
	TokenType   string       `json:"tokenType"`
	ExpiresIn   int64        `json:"expiresIn"`
	User        AuthUserView `json:"user"`
}

// AuthUserView 定义当前用户视图。
//
// 该视图只暴露前端会话建立后常用的展示字段，不包含密码哈希等敏感信息。
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
