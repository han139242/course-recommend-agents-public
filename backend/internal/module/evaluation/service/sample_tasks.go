package service

import (
	"github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/entity"
)

// BuildSampleTasks 构建示例评测任务（模拟学生数据和候选池）。
// 真实使用时应从数据库和 Neo4j 加载数据替换此函数。
func BuildSampleTasks() []entity.BenchmarkTask {
	history := []entity.AnswerRecord{
		{KnowledgeID: "sgd", CanonicalName: "随机梯度下降", MasteryScore: 45, CorrectRate: 0.55, AnsweredCount: 4, Status: "progressing"},
		{KnowledgeID: "gradient-descent", CanonicalName: "梯度下降", MasteryScore: 72, CorrectRate: 0.78, AnsweredCount: 8, Status: "solid"},
		{KnowledgeID: "loss-function", CanonicalName: "损失函数", MasteryScore: 85, CorrectRate: 0.90, AnsweredCount: 12, Status: "solid"},
		{KnowledgeID: "learning-rate", CanonicalName: "学习率", MasteryScore: 38, CorrectRate: 0.42, AnsweredCount: 3, Status: "struggling"},
		{KnowledgeID: "backpropagation", CanonicalName: "反向传播", MasteryScore: 60, CorrectRate: 0.65, AnsweredCount: 6, Status: "progressing"},
		{KnowledgeID: "matrix-ops", CanonicalName: "矩阵运算", MasteryScore: 90, CorrectRate: 0.95, AnsweredCount: 15, Status: "solid"},
		{KnowledgeID: "derivative", CanonicalName: "导数", MasteryScore: 88, CorrectRate: 0.92, AnsweredCount: 14, Status: "solid"},
	}

	candidates := []entity.CandidateNode{
		{KnowledgeID: "sgd", CanonicalName: "随机梯度下降", CourseTitle: "机器学习"},
		{KnowledgeID: "adam", CanonicalName: "Adam优化器", CourseTitle: "深度学习"},
		{KnowledgeID: "learning-rate", CanonicalName: "学习率", CourseTitle: "机器学习"},
		{KnowledgeID: "mini-batch", CanonicalName: "小批量训练", CourseTitle: "深度学习"},
		{KnowledgeID: "momentum", CanonicalName: "动量法", CourseTitle: "深度学习"},
		{KnowledgeID: "regularization", CanonicalName: "正则化", CourseTitle: "机器学习"},
		{KnowledgeID: "batch-norm", CanonicalName: "批归一化", CourseTitle: "深度学习"},
		{KnowledgeID: "convolution", CanonicalName: "卷积运算", CourseTitle: "深度学习"},
		{KnowledgeID: "rnn", CanonicalName: "循环神经网络", CourseTitle: "深度学习"},
		{KnowledgeID: "attention", CanonicalName: "注意力机制", CourseTitle: "大模型"},
	}

	groundTruth := []string{"adam", "momentum", "mini-batch", "learning-rate", "sgd"}

	prereqs := []entity.PrerequisiteEdge{
		{From: "derivative", To: "gradient-descent"},
		{From: "gradient-descent", To: "sgd"},
		{From: "sgd", To: "momentum"},
		{From: "momentum", To: "adam"},
		{From: "matrix-ops", To: "convolution"},
	}

	difficulties := map[string]float64{
		"sgd": 0.5, "adam": 0.7, "learning-rate": 0.4, "mini-batch": 0.4,
		"momentum": 0.6, "regularization": 0.6, "batch-norm": 0.6,
		"convolution": 0.6, "rnn": 0.7, "attention": 0.8,
	}

	return []entity.BenchmarkTask{{
		Name:           "optimization_chain",
		StudentHistory: history,
		CandidatePool:  candidates,
		GroundTruth:    groundTruth,
		Prerequisites:  prereqs,
		Difficulties:   difficulties,
	}}
}
