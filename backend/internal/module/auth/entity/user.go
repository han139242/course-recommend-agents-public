package entity

// User 表示当前阶段认证模块使用的用户模型。
type User struct {
	ID           int64
	Name         string
	Email        string
	Password     string
	Role         string
	Status       string
	Title        string
	StudentNo    string
	ClassID      *int64
	ClassName    string
	Target       string
	CreatedAtISO string
}
