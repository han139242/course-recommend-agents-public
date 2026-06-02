package service

import (
	"context"
	"sort"
	"strings"

	"github.com/han139242/course-recommend-agents/backend/internal/coursemeta"
	graphdto "github.com/han139242/course-recommend-agents/backend/internal/module/graph/dto"
	graphentity "github.com/han139242/course-recommend-agents/backend/internal/module/graph/entity"
	graphrepo "github.com/han139242/course-recommend-agents/backend/internal/module/graph/repository"
	"github.com/han139242/course-recommend-agents/backend/pkg/apperr"
)

// Service provides read-only graph query capabilities.
type Service struct {
	repo *graphrepo.Neo4jRepository
}

// New creates a graph service.
func New(repo *graphrepo.Neo4jRepository) *Service {
	return &Service{repo: repo}
}

// GetSubgraph loads a one-hop graph around a root node.
func (s *Service) GetSubgraph(ctx context.Context, query graphdto.SubgraphQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 12
	}

	nodes, edges, err := s.repo.LoadSubgraph(ctx, query.Root, limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetFullGraph loads a bounded graph snapshot for the homepage graph view.
func (s *Service) GetFullGraph(ctx context.Context, query graphdto.FullGraphQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 1000
	}

	nodes, edges, err := s.repo.LoadFullGraph(ctx, limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetCatalog loads lightweight graph metadata for search, navigation, and prerequisite lookup.
func (s *Service) GetCatalog(ctx context.Context) (*graphdto.CatalogResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	nodes, edges, err := s.repo.LoadCatalog(ctx)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildCatalogResponse(nodes, edges), nil
}

// GetSeeds loads seed nodes for the homepage graph canvas.
func (s *Service) GetSeeds(ctx context.Context, query graphdto.SeedQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	count := query.Count
	if count <= 0 {
		count = 14
	}

	nodes, edges, err := s.repo.LoadSeeds(ctx, count)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetNeighbors loads on-demand graph neighbors for a target node.
func (s *Service) GetNeighbors(ctx context.Context, query graphdto.NeighborQuery) (*graphdto.SubgraphResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	depth := query.Depth
	if depth <= 0 {
		depth = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	direction := query.Direction
	if direction == "" {
		direction = "both"
	}

	nodes, edges, err := s.repo.LoadNeighbors(ctx, query.NodeID, depth, direction, limit)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphResponse(nodes, edges), nil
}

// GetPrerequisiteChain loads a bounded prerequisite chain for a target knowledge node.
func (s *Service) GetPrerequisiteChain(ctx context.Context, query graphdto.PrerequisiteChainQuery) (*graphdto.PrerequisiteChainResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	depth := query.Depth
	if depth <= 0 {
		depth = 3
	}

	nodes, edges, orderedNodeIDs, err := s.repo.LoadPrerequisiteChain(ctx, query.Target, depth)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildPrerequisiteChainResponse(strings.TrimSpace(query.Target), nodes, edges, orderedNodeIDs), nil
}

// FindPath loads a shortest graph path between two knowledge nodes.
func (s *Service) FindPath(ctx context.Context, query graphdto.GraphPathQuery) (*graphdto.GraphPathResponse, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	maxDepth := query.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 5
	}

	nodes, edges, orderedNodeIDs, found, err := s.repo.FindShortestPath(ctx, query.From, query.To, maxDepth)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}

	return buildGraphPathResponse(nodes, edges, orderedNodeIDs, found), nil
}

// GetKnowledgeDetail loads detail information for a knowledge node.
func (s *Service) GetKnowledgeDetail(ctx context.Context, knowledgeID string) (*graphdto.KnowledgeDetailView, error) {
	if s.repo == nil || !s.repo.Enabled() {
		return nil, apperr.ErrDependencyUnavailable
	}

	detail, err := s.repo.LoadKnowledgeDetail(ctx, knowledgeID)
	if err != nil {
		return nil, apperr.ErrDependencyUnavailable.WithCause(err)
	}
	if detail == nil {
		return nil, apperr.ErrNotFound
	}

	return &graphdto.KnowledgeDetailView{
		ID:               detail.ID,
		RouteKnowledgeID: detail.RouteKnowledgeID,
		Name:             detail.Name,
		CanonicalName:    detail.CanonicalName,
		EntityType:       detail.EntityType,
		Description:      detail.Description,
		ChapterTitle: coursemeta.ResolveNodeCourseTitleFromHintsWithFallback(
			detail.CuratedSource,
			detail.SourcePath,
			"",
			"",
			"",
			detail.ChapterTitle,
			detail.Name,
			detail.CanonicalName,
			detail.Description,
			detail.ChapterTitle,
		),
		SourcePath:      detail.SourcePath,
		CuratedSource:   detail.CuratedSource,
		Summary:         detail.Summary,
		Prerequisites:   detail.Prerequisites,
		Related:         detail.Related,
		Scenarios:       detail.Scenarios,
		ParentTopics:    detail.ParentTopics,
		MasteryScore:    0,
		Status:          "not_started",
		MasteryText:     "当前暂无学习记录。",
		RecommendReason: "建议先从当前知识点的先修内容开始。",
	}, nil
}

func buildCatalogResponse(nodes []graphentity.Node, edges []graphentity.Edge) *graphdto.CatalogResponse {
	nodes, edges = normalizeStudentGraph(nodes, edges)
	response := &graphdto.CatalogResponse{
		Nodes:             make([]graphdto.CatalogNodeView, 0, len(nodes)),
		PrerequisiteEdges: make([]graphdto.PrerequisiteIndexEdgeView, 0, len(edges)),
	}

	for _, node := range nodes {
		response.Nodes = append(response.Nodes, graphdto.CatalogNodeView{
			ID:               node.ID,
			RouteKnowledgeID: node.RouteKnowledgeID,
			Name:             node.Name,
			CanonicalName:    node.CanonicalName,
			ChapterTitle: coursemeta.ResolveNodeCourseTitleFromHintsWithFallback(
				node.CuratedSource,
				node.SourcePath,
				"",
				"",
				"",
				node.ChapterTitle,
				node.Name,
				node.CanonicalName,
				node.Description,
				node.ChapterTitle,
			),
			EntityType:    node.EntityType,
			SourcePath:    node.SourcePath,
			CuratedSource: node.CuratedSource,
		})
	}

	for _, edge := range edges {
		if edge.SourceID == "" || edge.TargetID == "" {
			continue
		}
		sourceID := graphRouteID(nodes, edge.SourceID)
		targetID := graphRouteID(nodes, edge.TargetID)
		if sourceID == "" || targetID == "" || sourceID == targetID {
			continue
		}
		response.PrerequisiteEdges = append(response.PrerequisiteEdges, graphdto.PrerequisiteIndexEdgeView{
			SourceRouteKnowledgeID: sourceID,
			TargetRouteKnowledgeID: targetID,
		})
	}

	return response
}

func buildGraphResponse(nodes []graphentity.Node, edges []graphentity.Edge) *graphdto.SubgraphResponse {
	nodes, edges = normalizeStudentGraph(nodes, edges)
	response := &graphdto.SubgraphResponse{
		Nodes: make([]graphdto.GraphNodeView, 0, len(nodes)),
		Edges: make([]graphdto.GraphEdgeView, 0, len(edges)),
	}

	for _, node := range nodes {
		response.Nodes = append(response.Nodes, graphdto.GraphNodeView{
			ID:               node.ID,
			RouteKnowledgeID: node.RouteKnowledgeID,
			Name:             node.Name,
			CanonicalName:    node.CanonicalName,
			EntityType:       node.EntityType,
			Description:      node.Description,
			ChapterTitle: coursemeta.ResolveNodeCourseTitleFromHintsWithFallback(
				node.CuratedSource,
				node.SourcePath,
				"",
				"",
				"",
				node.ChapterTitle,
				node.Name,
				node.CanonicalName,
				node.Description,
				node.ChapterTitle,
			),
			SourcePath:    node.SourcePath,
			CuratedSource: node.CuratedSource,
		})
	}

	for _, edge := range edges {
		response.Edges = append(response.Edges, graphdto.GraphEdgeView{
			RelationID:   edge.RelationID,
			RelationType: edge.RelationType,
			SourceID:     edge.SourceID,
			TargetID:     edge.TargetID,
			SourceName:   edge.SourceName,
			TargetName:   edge.TargetName,
		})
	}

	return response
}

type graphAliasRule struct {
	RouteKnowledgeID string
	Name             string
	CanonicalName    string
	ChapterTitle     string
	Terms            []string
}

var graphAliasRules = []graphAliasRule{
	{RouteKnowledgeID: "llm-app", Name: "大语言模型", CanonicalName: "大语言模型", ChapterTitle: "大模型", Terms: []string{"large language model", "llm", "foundation model"}},
	{RouteKnowledgeID: "transformer", Name: "Transformer 架构", CanonicalName: "Transformer 架构", ChapterTitle: "大模型", Terms: []string{"transformer"}},
	{RouteKnowledgeID: "attention", Name: "注意力机制", CanonicalName: "注意力机制", ChapterTitle: "深度学习", Terms: []string{"attention mechanism", "self attention", "self-attention"}},
	{RouteKnowledgeID: "rag", Name: "检索增强生成 RAG", CanonicalName: "检索增强生成 RAG", ChapterTitle: "大模型", Terms: []string{"retrieval augmented generation", "retrieval-augmented generation", "rag"}},
	{RouteKnowledgeID: "agent-tool", Name: "Agent 与工具调用", CanonicalName: "Agent 与工具调用", ChapterTitle: "大模型", Terms: []string{"agent", "tool use", "multi agent", "multi-agent"}},
	{RouteKnowledgeID: "cnn", Name: "卷积神经网络 CNN", CanonicalName: "卷积神经网络 CNN", ChapterTitle: "深度学习", Terms: []string{"convolutional neural network", "cnn"}},
	{RouteKnowledgeID: "rnn", Name: "循环神经网络 RNN", CanonicalName: "循环神经网络 RNN", ChapterTitle: "深度学习", Terms: []string{"recurrent neural network", "long short term memory", "lstm"}},
	{RouteKnowledgeID: "nn-basic", Name: "神经网络基础", CanonicalName: "神经网络基础", ChapterTitle: "深度学习", Terms: []string{"artificial neural network", "deep neural network", "neural network"}},
	{RouteKnowledgeID: "decision-tree", Name: "决策树与集成方法", CanonicalName: "决策树与集成方法", ChapterTitle: "机器学习", Terms: []string{"decision tree", "ensemble learning"}},
	{RouteKnowledgeID: "linear-logistic", Name: "线性回归与逻辑回归", CanonicalName: "线性回归与逻辑回归", ChapterTitle: "机器学习", Terms: []string{"linear regression", "logistic regression"}},
	{RouteKnowledgeID: "gradient-descent", Name: "梯度下降", CanonicalName: "梯度下降", ChapterTitle: "机器学习", Terms: []string{"gradient descent", "optimization"}},
	{RouteKnowledgeID: "loss-function", Name: "损失函数", CanonicalName: "损失函数", ChapterTitle: "机器学习", Terms: []string{"softmax function", "sigmoid function", "relu function", "activation function"}},
	{RouteKnowledgeID: "knowledge-graph", Name: "知识图谱构建与查询", CanonicalName: "知识图谱构建与查询", ChapterTitle: "知识图谱", Terms: []string{"knowledge graph", "graph reasoning"}},
	{RouteKnowledgeID: "sentiment", Name: "自然语言处理", CanonicalName: "自然语言处理", ChapterTitle: "行业实践", Terms: []string{"natural language processing", "nlp"}},
}

var graphNameOverrides = map[string]string{
	"a2c":                  "优势 Actor-Critic（A2C）",
	"a3c":                  "异步优势 Actor-Critic（A3C）",
	"actor critic":         "演员-评论家算法（Actor-Critic）",
	"actor-critic":         "演员-评论家算法（Actor-Critic）",
	"adaboost":             "自适应提升算法（AdaBoost）",
	"adagrad":              "自适应梯度算法（AdaGrad）",
	"adam":                 "Adam 优化器",
	"adamw":                "AdamW 优化器",
	"adapter":              "适配器微调（Adapter）",
	"ai for science":       "AI for Science 科学智能",
	"auc":                  "AUC 评估指标",
	"bagging":              "装袋集成（Bagging）",
	"bleu":                 "BLEU 机器翻译指标",
	"boosting":             "提升集成（Boosting）",
	"constitutional ai":    "宪法式 AI（Constitutional AI）",
	"ctde":                 "集中训练分散执行（CTDE）",
	"dataframe":            "数据表对象（DataFrame）",
	"dbscan":               "密度聚类算法（DBSCAN）",
	"deep rl":              "深度强化学习（Deep RL）",
	"double dqn":           "双重 DQN（Double DQN）",
	"dropout":              "随机失活（Dropout）",
	"dueling dqn":          "竞争网络 DQN（Dueling DQN）",
	"elastic net":          "弹性网络回归（Elastic Net）",
	"eu ai act":            "欧盟人工智能法案（EU AI Act）",
	"expected sarsa":       "期望 SARSA（Expected SARSA）",
	"gil":                  "全局解释器锁（GIL）",
	"gradient descent":     "梯度下降",
	"large language model": "大语言模型",
	"ml pipeline":          "机器学习流水线（ML Pipeline）",
	"mlops":                "机器学习运维（MLOps）",
	"momentum":             "动量优化（Momentum）",
	"ndarray":              "多维数组（ndarray）",
	"noisynet":             "噪声网络（NoisyNet）",
	"pagedattention":       "分页注意力（PagedAttention）",
	"plan and execute":     "规划执行范式（Plan-and-Execute）",
	"plan-and-execute":     "规划执行范式（Plan-and-Execute）",
	"ppo":                  "近端策略优化（PPO）",
	"p tuning":             "提示调优（P-Tuning）",
	"p-tuning":             "提示调优（P-Tuning）",
	"reinforce":            "REINFORCE 策略梯度",
	"rlaif":                "AI 反馈强化学习（RLAIF）",
	"rmsprop":              "RMSProp 优化器",
	"rouge":                "ROUGE 文本生成指标",
	"sac":                  "软 Actor-Critic（SAC）",
	"sarsa":                "SARSA 强化学习算法",
	"self consistency":     "自洽推理（Self-Consistency）",
	"self-consistency":     "自洽推理（Self-Consistency）",
	"series":               "序列对象（Series）",
	"shap":                 "SHAP 可解释性方法",
	"stable diffusion":     "稳定扩散模型（Stable Diffusion）",
	"stacking":             "堆叠集成（Stacking）",
	"t sne":                "t-SNE 降维可视化",
	"t-sne":                "t-SNE 降维可视化",
	"tf idf":               "TF-IDF 词项权重",
	"tf-idf":               "TF-IDF 词项权重",
	"transformer":          "Transformer 架构",
	"trpo":                 "信赖域策略优化（TRPO）",
	"warmup":               "学习率预热（Warmup）",
	"world model":          "世界模型（World Model）",
	"zero":                 "ZeRO 分布式优化",
}

func normalizeStudentGraph(nodes []graphentity.Node, edges []graphentity.Edge) ([]graphentity.Node, []graphentity.Edge) {
	normalizedNodes := make([]graphentity.Node, 0, len(nodes))
	originalToGroup := make(map[string]string, len(nodes))
	groupToNode := make(map[string]graphentity.Node, len(nodes))
	groupHasExternal := make(map[string]bool, len(nodes))
	groupHasLocal := make(map[string]bool, len(nodes))

	for _, raw := range nodes {
		node, groupKey, keep := normalizeStudentGraphNode(raw)
		if !keep || node.ID == "" {
			continue
		}
		originalToGroup[raw.ID] = groupKey
		existing, exists := groupToNode[groupKey]
		if !exists || preferGraphNode(node, existing) {
			groupToNode[groupKey] = node
		}
		if isExternalOntologyNode(raw) {
			groupHasExternal[groupKey] = true
		} else {
			groupHasLocal[groupKey] = true
		}
	}

	groupToID := make(map[string]string, len(groupToNode))
	for groupKey, node := range groupToNode {
		groupToID[groupKey] = node.ID
	}

	edgeSeen := map[string]struct{}{}
	normalizedEdges := make([]graphentity.Edge, 0, len(edges))
	connected := map[string]struct{}{}
	for _, edge := range edges {
		sourceGroup := originalToGroup[edge.SourceID]
		targetGroup := originalToGroup[edge.TargetID]
		sourceID := groupToID[sourceGroup]
		targetID := groupToID[targetGroup]
		if sourceID == "" || targetID == "" || sourceID == targetID {
			continue
		}
		relationType := strings.TrimSpace(edge.RelationType)
		if relationType == "" {
			relationType = "RELATED_TO"
		}
		key := sourceID + ":" + relationType + ":" + targetID
		if _, exists := edgeSeen[key]; exists {
			continue
		}
		edgeSeen[key] = struct{}{}
		connected[sourceID] = struct{}{}
		connected[targetID] = struct{}{}
		normalizedEdges = append(normalizedEdges, graphentity.Edge{
			RelationID:   firstNonEmpty(edge.RelationID, key),
			RelationType: relationType,
			SourceID:     sourceID,
			TargetID:     targetID,
			SourceName:   graphDisplayName(groupToNode[sourceGroup]),
			TargetName:   graphDisplayName(groupToNode[targetGroup]),
		})
	}

	for groupKey, node := range groupToNode {
		if groupHasExternal[groupKey] && !groupHasLocal[groupKey] {
			if _, ok := connected[node.ID]; !ok {
				continue
			}
		}
		normalizedNodes = append(normalizedNodes, node)
	}
	sort.Slice(normalizedNodes, func(i, j int) bool {
		return normalizedNodes[i].ID < normalizedNodes[j].ID
	})
	sort.Slice(normalizedEdges, func(i, j int) bool {
		return normalizedEdges[i].RelationID < normalizedEdges[j].RelationID
	})
	return normalizedNodes, normalizedEdges
}

func normalizeStudentGraphNode(node graphentity.Node) (graphentity.Node, string, bool) {
	next := node
	next.ID = strings.TrimSpace(next.ID)
	next.RouteKnowledgeID = firstNonEmpty(next.RouteKnowledgeID, next.ID)
	next.Name = firstNonEmpty(cleanExternalOntologyName(next.Name), cleanExternalOntologyName(next.CanonicalName), next.ID)
	next.CanonicalName = firstNonEmpty(cleanExternalOntologyName(next.CanonicalName), next.Name)

	alias, matched := matchGraphAlias(next)
	external := isExternalOntologyNode(node)
	if external && !matched {
		return graphentity.Node{}, "", false
	}
	if matched {
		next.RouteKnowledgeID = alias.RouteKnowledgeID
		next.Name = alias.Name
		next.CanonicalName = alias.CanonicalName
		if strings.TrimSpace(next.ChapterTitle) == "" || external {
			next.ChapterTitle = alias.ChapterTitle
		}
		if strings.TrimSpace(next.EntityType) == "" {
			next.EntityType = "concept"
		}
	}
	next = localizeGraphNodeName(next)

	groupKey := graphCanonicalKey(next)
	return next, groupKey, true
}

func matchGraphAlias(node graphentity.Node) (graphAliasRule, bool) {
	candidates := []string{
		node.ID,
		node.RouteKnowledgeID,
		node.Name,
		node.CanonicalName,
		node.Description,
	}
	normalizedCandidates := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		normalizedCandidates = append(normalizedCandidates, normalizeGraphAliasText(candidate))
	}
	for _, rule := range graphAliasRules {
		for _, term := range rule.Terms {
			needle := normalizeGraphAliasText(term)
			if needle == "" {
				continue
			}
			for _, candidate := range normalizedCandidates {
				if candidate == needle || strings.Contains(candidate, needle) {
					return rule, true
				}
			}
		}
	}
	return graphAliasRule{}, false
}

func graphCanonicalKey(node graphentity.Node) string {
	if id := strings.TrimSpace(node.RouteKnowledgeID); id != "" && !strings.HasPrefix(strings.ToLower(id), "aio:") && !looksGeneratedGraphID(id) {
		return strings.ToLower(id)
	}
	return normalizeGraphAliasText(firstNonEmpty(node.CanonicalName, node.Name, node.ID))
}

func preferGraphNode(candidate graphentity.Node, existing graphentity.Node) bool {
	if isExternalOntologyNode(existing) != isExternalOntologyNode(candidate) {
		return !isExternalOntologyNode(candidate)
	}
	if strings.TrimSpace(candidate.Description) != "" && strings.TrimSpace(existing.Description) == "" {
		return true
	}
	if strings.TrimSpace(candidate.ChapterTitle) != "" && strings.TrimSpace(existing.ChapterTitle) == "" {
		return true
	}
	return len(candidate.ID) < len(existing.ID)
}

func graphRouteID(nodes []graphentity.Node, nodeID string) string {
	for _, node := range nodes {
		if node.ID == nodeID {
			return firstNonEmpty(node.RouteKnowledgeID, node.ID)
		}
	}
	return nodeID
}

func graphDisplayName(node graphentity.Node) string {
	return firstNonEmpty(node.CanonicalName, node.Name, node.ID)
}

func cleanExternalOntologyName(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "AIO:")
	trimmed = strings.TrimPrefix(trimmed, "aio:")
	return strings.TrimSpace(trimmed)
}

func localizeGraphNodeName(node graphentity.Node) graphentity.Node {
	next := node
	if localized := localizedGraphName(firstNonEmpty(next.CanonicalName, next.Name), next.ChapterTitle, next.EntityType); localized != "" {
		if shouldReplaceGraphDisplayName(next.Name) || normalizeGraphAliasText(next.Name) == normalizeGraphAliasText(next.CanonicalName) {
			next.Name = localized
		}
		if shouldReplaceGraphDisplayName(next.CanonicalName) {
			next.CanonicalName = localized
		}
	}
	return next
}

func localizedGraphName(value string, chapterTitle string, entityType string) string {
	name := cleanExternalOntologyName(value)
	if name == "" || containsCJK(name) {
		return ""
	}
	key := normalizeGraphAliasText(name)
	if override := graphNameOverrides[key]; override != "" {
		return override
	}
	if !isPlainEnglishGraphName(name) {
		return ""
	}
	return name + "（" + graphNameSuffix(chapterTitle, entityType) + "）"
}

func shouldReplaceGraphDisplayName(value string) bool {
	return strings.TrimSpace(value) == "" || isPlainEnglishGraphName(cleanExternalOntologyName(value))
}

func graphNameSuffix(chapterTitle string, entityType string) string {
	chapter := strings.ToLower(strings.TrimSpace(chapterTitle))
	switch {
	case strings.Contains(chapter, "python"):
		return "Python 知识点"
	case strings.Contains(chapter, "机器学习"):
		return "机器学习方法"
	case strings.Contains(chapter, "深度学习"):
		return "深度学习模型"
	case strings.Contains(chapter, "自然语言"):
		return "自然语言处理技术"
	case strings.Contains(chapter, "计算机视觉"):
		return "计算机视觉方法"
	case strings.Contains(chapter, "强化学习"):
		return "强化学习算法"
	case strings.Contains(chapter, "知识图谱"):
		return "知识图谱技术"
	case strings.Contains(chapter, "大模型"):
		return "大模型技术"
	case strings.Contains(chapter, "ai工程") || strings.Contains(chapter, "ai 工程"):
		return "AI 工程工具"
	case strings.Contains(chapter, "ai伦理") || strings.Contains(chapter, "ai 伦理"):
		return "AI 伦理主题"
	case strings.Contains(chapter, "ai for science"):
		return "科学智能技术"
	case strings.Contains(chapter, "人工智能"):
		return "人工智能方法"
	}
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case "algorithm":
		return "算法"
	case "model":
		return "模型"
	case "tool":
		return "工具"
	case "method":
		return "方法"
	default:
		return "课程知识点"
	}
}

func isExternalOntologyNode(node graphentity.Node) bool {
	joined := strings.ToLower(strings.Join([]string{
		node.ID,
		node.RouteKnowledgeID,
		node.Name,
		node.CanonicalName,
		node.SourcePath,
		node.CuratedSource,
		node.ChapterTitle,
	}, " "))
	return strings.Contains(joined, "aio:") || strings.Contains(joined, "aio ontology")
}

func normalizeGraphAliasText(value string) string {
	normalized := strings.ToLower(cleanExternalOntologyName(value))
	replacer := strings.NewReplacer(
		"_", " ",
		"-", " ",
		":", " ",
		"（", " ",
		"）", " ",
		"(", " ",
		")", " ",
		"/", " ",
		",", " ",
		".", " ",
	)
	normalized = replacer.Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func containsCJK(value string) bool {
	for _, char := range value {
		if char >= '\u4e00' && char <= '\u9fff' {
			return true
		}
	}
	return false
}

func isPlainEnglishGraphName(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || containsCJK(trimmed) {
		return false
	}
	hasLetter := false
	for _, char := range trimmed {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
			continue
		}
		if (char >= '0' && char <= '9') || strings.ContainsRune(" -_+./'&", char) {
			continue
		}
		return false
	}
	return hasLetter
}

func looksGeneratedGraphID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 16 && len(trimmed) != 32 {
		return false
	}
	for _, char := range trimmed {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func buildPrerequisiteChainResponse(target string, nodes []graphentity.Node, edges []graphentity.Edge, orderedNodeIDs []string) *graphdto.PrerequisiteChainResponse {
	graph := buildGraphResponse(nodes, edges)
	return &graphdto.PrerequisiteChainResponse{
		Target:         target,
		Nodes:          graph.Nodes,
		Edges:          graph.Edges,
		OrderedNodeIDs: normalizeOrderedNodeIDs(orderedNodeIDs),
	}
}

func buildGraphPathResponse(nodes []graphentity.Node, edges []graphentity.Edge, orderedNodeIDs []string, found bool) *graphdto.GraphPathResponse {
	graph := buildGraphResponse(nodes, edges)
	return &graphdto.GraphPathResponse{
		Found:          found,
		Nodes:          graph.Nodes,
		Edges:          graph.Edges,
		OrderedNodeIDs: normalizeOrderedNodeIDs(orderedNodeIDs),
	}
}

func normalizeOrderedNodeIDs(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
