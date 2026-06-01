package service

import (
	"context"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/module/evaluation/entity"
	recommendservice "github.com/han139242/course-recommend-agents/backend/internal/module/recommend/service"
)

// BaselineFunc defines a baseline strategy.
type BaselineFunc func(task entity.BenchmarkTask) []string

// BaselineRegistry holds all registered baselines.
type BaselineRegistry struct {
	entries map[string]BaselineFunc
	order   []string
}

// NewBaselineRegistry creates a registry with built-in baselines.
func NewBaselineRegistry() *BaselineRegistry {
	r := &BaselineRegistry{entries: make(map[string]BaselineFunc)}
	r.Register("random", RandomBaseline)
	r.Register("course_order", CourseOrderBaseline)
	r.Register("prerequisite_shortest", PrerequisiteShortestPathBaseline)
	r.Register("weakness_first", WeaknessFirstBaseline)
	return r
}

func (r *BaselineRegistry) Register(name string, fn BaselineFunc) {
	if r.entries == nil {
		r.entries = make(map[string]BaselineFunc)
	}
	r.entries[name] = fn
	r.order = append(r.order, name)
}

func (r *BaselineRegistry) List() []string {
	return append([]string(nil), r.order...)
}

func (r *BaselineRegistry) Run(name string, task entity.BenchmarkTask) []string {
	fn, ok := r.entries[name]
	if !ok {
		return nil
	}
	return fn(task)
}

// RandomBaseline 随机打乱候选池。
func RandomBaseline(task entity.BenchmarkTask) []string {
	ids := candidateIDs(task.CandidatePool)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	return ids
}

// CourseOrderBaseline 按课程教学顺序排列。
func CourseOrderBaseline(task entity.BenchmarkTask) []string {
	courseOrder := map[string]int{
		"高等数学": 1, "线性代数": 2, "概率与统计": 3,
		"数据结构": 4, "Python编程": 5,
		"人工智能导论": 6, "机器学习": 7, "深度学习": 8,
		"自然语言处理": 9, "计算机视觉": 10, "强化学习": 11,
		"知识图谱": 12, "大模型": 13,
		"高级大模型应用开发": 14, "AI工程": 15,
		"AI基础设施与高性能智能计算": 16,
		"AI for Science": 17, "具身智能与世界模型": 18,
		"AI安全攻防": 19, "AI伦理": 20, "其他": 21,
	}

	type indexed struct {
		id    string
		order int
	}
	items := make([]indexed, len(task.CandidatePool))
	for i, c := range task.CandidatePool {
		order, ok := courseOrder[c.CourseTitle]
		if !ok {
			order = 99
		}
		items[i] = indexed{id: c.KnowledgeID, order: order}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].order < items[j].order
	})

	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.id
	}
	return ids
}

// PrerequisiteShortestPathBaseline 按先修最短路径排列。
func PrerequisiteShortestPathBaseline(task entity.BenchmarkTask) []string {
	edgeMap := make(map[string][]string, len(task.Prerequisites))
	for _, e := range task.Prerequisites {
		edgeMap[e.From] = append(edgeMap[e.From], e.To)
	}

	distance := make(map[string]int, len(task.CandidatePool))

	type queueItem struct {
		id   string
		dist int
	}
	queue := make([]queueItem, 0)
	for _, record := range task.StudentHistory {
		if record.MasteryScore >= 60 && record.AnsweredCount > 0 {
			queue = append(queue, queueItem{id: record.KnowledgeID, dist: 0})
		}
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if _, exists := distance[item.id]; exists {
			continue
		}
		distance[item.id] = item.dist

		for _, next := range edgeMap[item.id] {
			if _, exists := distance[next]; !exists {
				queue = append(queue, queueItem{id: next, dist: item.dist + 1})
			}
		}
	}

	type bfsDist struct {
		id   string
		dist int
	}
	bfsItems := make([]bfsDist, len(task.CandidatePool))
	for i, c := range task.CandidatePool {
		dist, ok := distance[c.KnowledgeID]
		if !ok {
			dist = 999
		}
		bfsItems[i] = bfsDist{id: c.KnowledgeID, dist: dist}
	}
	sort.SliceStable(bfsItems, func(i, j int) bool {
		if bfsItems[i].dist == bfsItems[j].dist {
			return bfsItems[i].id < bfsItems[j].id
		}
		return bfsItems[i].dist < bfsItems[j].dist
	})

	ids := make([]string, len(bfsItems))
	for i, s := range bfsItems {
		ids[i] = s.id
	}
	return ids
}

// WeaknessFirstBaseline 薄弱点优先排列。
func WeaknessFirstBaseline(task entity.BenchmarkTask) []string {
	masteryMap := make(map[string]float64, len(task.StudentHistory))
	for _, r := range task.StudentHistory {
		masteryMap[r.KnowledgeID] = r.MasteryScore
	}

	type masteryScored struct {
		id      string
		mastery float64
	}
	items := make([]masteryScored, len(task.CandidatePool))
	for i, c := range task.CandidatePool {
		m, ok := masteryMap[c.KnowledgeID]
		if !ok {
			m = 50
		}
		items[i] = masteryScored{id: c.KnowledgeID, mastery: m}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].mastery == items[j].mastery {
			return items[i].id < items[j].id
		}
		return items[i].mastery < items[j].mastery
	})

	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.id
	}
	return ids
}

func candidateIDs(pool []entity.CandidateNode) []string {
	ids := make([]string, len(pool))
	for i, c := range pool {
		ids[i] = strings.TrimSpace(c.KnowledgeID)
	}
	return ids
}

// KTKGPathBaseline uses the unified scorer to rank candidates.
func KTKGPathBaseline(scorer *recommendservice.Scorer) BaselineFunc {
	return func(task entity.BenchmarkTask) []string {
		if scorer == nil {
			return candidateIDs(task.CandidatePool)
		}

		stateLookup := make(map[string]recommendservice.StudentKnowledgeState, len(task.StudentHistory))
		for _, r := range task.StudentHistory {
			stateLookup[r.KnowledgeID] = recommendservice.StudentKnowledgeState{
				KnowledgeID:    r.KnowledgeID,
				CanonicalName:  r.CanonicalName,
				MasteryScore:   r.MasteryScore,
				CorrectRate:    r.CorrectRate,
				AnsweredCount:  r.AnsweredCount,
				LastAnsweredAt: r.LastAnsweredAt,
				Status:         r.Status,
			}
		}

		type ktScored struct {
			id    string
			score float64
		}

		results := make([]ktScored, 0, len(task.CandidatePool))
		for _, c := range task.CandidatePool {
			state, ok := stateLookup[c.KnowledgeID]
			if !ok {
				state = recommendservice.StudentKnowledgeState{
					KnowledgeID:   c.KnowledgeID,
					CanonicalName: c.CanonicalName,
					MasteryScore:  0,
					CorrectRate:   0,
					AnsweredCount: 0,
					Status:        "not_started",
				}
			}

			input := recommendservice.NewScoreInput(
				c.KnowledgeID, c.CanonicalName,
				state.MasteryScore, state.CorrectRate, state.AnsweredCount,
				state.LastAnsweredAt, state.Status,
				stateLookup,
				"", "", c.CourseTitle,
				"RELATED_TO",
				0,
			)

			s, _ := scorer.ScoreCandidate(context.Background(), input)
			results = append(results, ktScored{id: c.KnowledgeID, score: s})
		}

		sort.SliceStable(results, func(i, j int) bool {
			if results[i].score == results[j].score {
				return results[i].id < results[j].id
			}
			return results[i].score > results[j].score
		})

		ids := make([]string, len(results))
		for i, r := range results {
			ids[i] = r.id
		}
		return ids
	}
}
