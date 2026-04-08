package service

import (
	competencyentity "github.com/han139242/course-recommend-agents/backend/internal/module/competency/entity"
	authentity "github.com/han139242/course-recommend-agents/backend/internal/module/auth/entity"
)

// UserToProfile 将认证用户实体转换为能力画像用户视图。
func UserToProfile(user authentity.User) competencyentity.UserProfile {
	return competencyentity.UserProfile{ID: user.ID, Name: user.Name, Role: user.Role, ClassID: user.ClassID, ClassName: user.ClassName, Target: user.Target}
}
