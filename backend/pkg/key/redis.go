package key

import "fmt"

const redisKeyVersion = "v1"

// UserKnowledgeProgress 返回学生知识点掌握度向量 key。
func UserKnowledgeProgress(studentID int64) string {
	return fmt.Sprintf("uid:%d:kp:%s", studentID, redisKeyVersion)
}

// UserKnowledgeProgressAnswered 返回学生知识点作答次数 key。
func UserKnowledgeProgressAnswered(studentID int64) string {
	return fmt.Sprintf("uid:%d:kp:answered:%s", studentID, redisKeyVersion)
}

// UserKnowledgeProgressCorrect 返回学生知识点答对次数 key。
func UserKnowledgeProgressCorrect(studentID int64) string {
	return fmt.Sprintf("uid:%d:kp:correct:%s", studentID, redisKeyVersion)
}

// UserKnowledgeProgressLastAnswered 返回学生知识点最近作答时间 key。
func UserKnowledgeProgressLastAnswered(studentID int64) string {
	return fmt.Sprintf("uid:%d:kp:last:%s", studentID, redisKeyVersion)
}

// UserCompetencyProfile 返回学生能力画像缓存 key。
func UserCompetencyProfile(studentID int64) string {
	return fmt.Sprintf("uid:%d:competency:%s", studentID, redisKeyVersion)
}

// UserRecommendPath 返回学生推荐路径缓存 key。
func UserRecommendPath(studentID int64, targetID string, scene string, limit int) string {
	return fmt.Sprintf("uid:%d:recommend:%s:%s:%d:%s", studentID, targetID, scene, limit, redisKeyVersion)
}
