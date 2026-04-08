package dto

// LoginRequest 定义登录请求。
//
// 字段约束尽量前置在绑定阶段，避免无效请求进入业务层后再触发不必要的仓储或密码校验开销。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}
