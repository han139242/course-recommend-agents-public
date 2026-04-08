package ctxmeta

import (
	"context"
	"strconv"
)

const roleKey contextKey = "role"

// WithRole 向上下文写入角色信息。
// 角色信息用于后续业务层快速判断权限边界。
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// Role 从上下文读取角色信息。
func Role(ctx context.Context) string {
	value, _ := ctx.Value(roleKey).(string)
	return value
}

// CurrentUserID 从上下文读取当前用户 ID。
// 该方法对解析失败返回 false，调用方可据此决定是否视为未登录态。
func CurrentUserID(ctx context.Context) (int64, bool) {
	userID := UserID(ctx)
	if userID == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
