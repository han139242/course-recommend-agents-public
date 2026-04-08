package repository

const (
	// UserStatusActive 表示用户处于可登录状态。
	//
	// 认证链路只放行该状态，避免被停用账号继续获取新令牌。
	UserStatusActive = "active"
	// UserStatusDisabled 表示用户已被禁用。
	//
	// 即使账号历史上签发过令牌，服务端后续也应基于最新状态拒绝访问。
	UserStatusDisabled = "disabled"
)
