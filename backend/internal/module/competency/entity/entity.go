package entity

// UserProfile 表示能力画像所需的当前用户信息。
type UserProfile struct {
	ID        int64
	Name      string
	Role      string
	ClassID   *int64
	ClassName *string
	Target    string
}
