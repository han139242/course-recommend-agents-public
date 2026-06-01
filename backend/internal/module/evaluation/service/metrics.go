package service

import (
	"math"

	"github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/entity"
)

// ComputeHitAtK 计算 Hit@K：推荐列表中前 K 个中包含至少一个真值项的比例。
func ComputeHitAtK(recommended []string, groundTruth []string, k int) float64 {
	if len(recommended) == 0 || len(groundTruth) == 0 {
		return 0
	}
	truthSet := toStringSet(groundTruth)
	limit := k
	if limit > len(recommended) {
		limit = len(recommended)
	}
	for i := 0; i < limit; i++ {
		if truthSet[recommended[i]] {
			return 1.0
		}
	}
	return 0.0
}

// ComputeNDCGAtK 计算 NDCG@K：折损累积增益。
func ComputeNDCGAtK(recommended []string, groundTruth []string, k int) float64 {
	if len(recommended) == 0 || len(groundTruth) == 0 {
		return 0
	}
	truthSet := toStringSet(groundTruth)

	limit := k
	if limit > len(recommended) {
		limit = len(recommended)
	}

	dcg := 0.0
	for i := 0; i < limit; i++ {
		if truthSet[recommended[i]] {
			dcg += 1.0 / math.Log2(float64(i)+2)
		}
	}

	idealLen := k
	if idealLen > len(groundTruth) {
		idealLen = len(groundTruth)
	}
	idcg := 0.0
	for i := 0; i < idealLen; i++ {
		idcg += 1.0 / math.Log2(float64(i)+2)
	}

	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

// ComputeMRR 计算 MRR（Mean Reciprocal Rank）：第一个相关项的倒数排名。
func ComputeMRR(recommended []string, groundTruth []string) float64 {
	if len(recommended) == 0 || len(groundTruth) == 0 {
		return 0
	}
	truthSet := toStringSet(groundTruth)
	for i, id := range recommended {
		if truthSet[id] {
			return 1.0 / float64(i+1)
		}
	}
	return 0.0
}

// ComputePrerequisiteViolationRate 计算先修违规率。
// 对推荐路径中每一对相邻项，若 (后, 前) 是合法先修边则违规。
func ComputePrerequisiteViolationRate(path []string, edges []entity.PrerequisiteEdge) float64 {
	if len(path) < 2 || len(edges) == 0 {
		return 0
	}
	edgeSet := make(map[string]bool, len(edges))
	for _, e := range edges {
		edgeSet[e.From+"||"+e.To] = true
	}
	violations := 0
	for i := 0; i < len(path)-1; i++ {
		if edgeSet[path[i+1]+"||"+path[i]] {
			violations++
		}
	}
	return float64(violations) / float64(len(path)-1)
}

// ComputeDifficultySmoothness 计算难度平滑度：相邻项难度差的均方根。
// 值越小越平滑，返回 [0,1] 归一化值（越低越好）。
func ComputeDifficultySmoothness(path []string, difficulties map[string]float64) float64 {
	if len(path) < 2 || len(difficulties) == 0 {
		return 1.0
	}
	sumSqDiff := 0.0
	count := 0
	for i := 0; i < len(path)-1; i++ {
		d1, ok1 := difficulties[path[i]]
		d2, ok2 := difficulties[path[i+1]]
		if ok1 && ok2 {
			diff := d1 - d2
			sumSqDiff += diff * diff
			count++
		}
	}
	if count == 0 {
		return 1.0
	}
	rmse := math.Sqrt(sumSqDiff / float64(count))
	// 归一化：假设难度差最大为 1.0
	return 1.0 - clamp(rmse, 0, 1.0)
}

func toStringSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func clamp(v, min, max float64) float64 {
	if math.IsNaN(v) {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
