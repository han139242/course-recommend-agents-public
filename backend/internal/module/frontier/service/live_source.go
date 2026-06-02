package service

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	frontierdto "github.com/han139242/course-recommend-agents/backend/internal/module/frontier/dto"
)

const (
	defaultArxivEndpoint    = "http://export.arxiv.org/api/query"
	defaultGitHubEndpoint   = "https://api.github.com/search/repositories"
	defaultOpenAlexEndpoint = "https://api.openalex.org/works"
	defaultCacheTTL         = 7 * 24 * time.Hour
	defaultLiveLimit        = 12
)

type liveSource struct {
	client           *http.Client
	arxivEndpoint    string
	gitHubEndpoint   string
	openAlexEndpoint string
	cacheTTL         time.Duration
	now              func() time.Time
	mu               sync.Mutex
	cache            *libraryCache
}

type libraryCache struct {
	response  *frontierdto.LibraryResponse
	expiresAt time.Time
}

type sourceConfig struct {
	client           *http.Client
	arxivEndpoint    string
	gitHubEndpoint   string
	openAlexEndpoint string
	cacheTTL         time.Duration
	now              func() time.Time
}

// NewLiveSource 创建真实在线前沿资料来源。
func NewLiveSource() LiveSource {
	return newLiveSource(sourceConfig{})
}

func newLiveSource(config sourceConfig) *liveSource {
	client := config.client
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	arxivEndpoint := strings.TrimSpace(config.arxivEndpoint)
	if arxivEndpoint == "" {
		arxivEndpoint = defaultArxivEndpoint
	}
	gitHubEndpoint := strings.TrimSpace(config.gitHubEndpoint)
	if gitHubEndpoint == "" {
		gitHubEndpoint = defaultGitHubEndpoint
	}
	openAlexEndpoint := strings.TrimSpace(config.openAlexEndpoint)
	if openAlexEndpoint == "" {
		openAlexEndpoint = defaultOpenAlexEndpoint
	}
	cacheTTL := config.cacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}
	now := config.now
	if now == nil {
		now = time.Now
	}
	return &liveSource{
		client:           client,
		arxivEndpoint:    arxivEndpoint,
		gitHubEndpoint:   gitHubEndpoint,
		openAlexEndpoint: openAlexEndpoint,
		cacheTTL:         cacheTTL,
		now:              now,
	}
}

func (s *liveSource) ListLibrary(ctx context.Context, limit int) (*frontierdto.LibraryResponse, error) {
	if limit <= 0 {
		limit = defaultLiveLimit
	}
	limit = minInt(limit, defaultLiveLimit)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if s.cache != nil && now.Before(s.cache.expiresAt) {
		return limitLibraryResponse(s.cache.response, limit), nil
	}

	response, err := s.fetchFreshLibrary(ctx, limit)
	if err != nil {
		return notReadyLiveLibraryResponse(err), nil
	}
	response.GeneratedAt = now.Format(time.RFC3339)
	response.NextRefreshAt = now.Add(s.cacheTTL).Format(time.RFC3339)
	response.RefreshInterval = "7d"

	s.cache = &libraryCache{
		response:  cloneLibraryResponse(response),
		expiresAt: now.Add(s.cacheTTL),
	}
	return limitLibraryResponse(response, limit), nil
}

func (s *liveSource) GetBriefing(ctx context.Context, knowledgeID string, knowledgeName string) (*frontierdto.BriefingResponse, error) {
	knowledge := normalizeBriefingKnowledge(knowledgeID, knowledgeName)
	library, err := s.ListLibrary(ctx, defaultLiveLimit)
	if err != nil {
		return nil, err
	}
	if library == nil || len(library.Items) == 0 {
		return &frontierdto.BriefingResponse{
			KnowledgeID:   normalizeBriefingID(knowledgeID, knowledgeName),
			Knowledge:     knowledge,
			Status:        "not_ready",
			Message:       "暂时没有可匹配的真实前沿资料。",
			Topics:        []frontierdto.TopicView{},
			Papers:        []frontierdto.PaperView{},
			GeneratedAt:   "",
			DigestSummary: "",
		}, nil
	}

	matches := filterItemsByKnowledge(library.Items, knowledge)
	if len(matches) == 0 {
		matches = library.Items[:minInt(3, len(library.Items))]
	}
	topics := uniqueTopics(matches, 5)
	papers := make([]frontierdto.PaperView, 0, minInt(4, len(matches)))
	for _, item := range matches[:minInt(4, len(matches))] {
		papers = append(papers, frontierdto.PaperView{
			PaperID:     item.PaperID,
			Title:       item.Title,
			Venue:       item.Venue,
			Year:        item.Year,
			URL:         item.URL,
			Authors:     item.Authors,
			PublishedAt: item.PublishedAt,
		})
	}

	return &frontierdto.BriefingResponse{
		KnowledgeID: normalizeBriefingID(knowledgeID, knowledgeName),
		Status:      "ready",
		Knowledge:   knowledge,
		Message:     "已从 arXiv 与 GitHub 公开来源载入真实前沿资料。",
		DigestSummary: fmt.Sprintf(
			"围绕“%s”匹配到 %d 条最新公开资料，来源包含 arXiv 论文和 GitHub 开源更新；这些内容按周缓存，适合作为课程知识点的课后延伸阅读。",
			knowledge,
			len(matches),
		),
		Topics:      topics,
		Papers:      papers,
		GeneratedAt: library.GeneratedAt,
	}, nil
}

func (s *liveSource) fetchFreshLibrary(ctx context.Context, limit int) (*frontierdto.LibraryResponse, error) {
	paperLimit := minInt(limit, 8)
	gitHubLimit := minInt(4, maxInt(limit-paperLimit, 4))

	items := make([]frontierdto.LibraryItemView, 0, limit)
	arxivItems, arxivErr := s.fetchArxivItems(ctx, minInt(paperLimit, 4))
	items = append(items, arxivItems...)

	if len(items) < paperLimit {
		openAlexItems, _ := s.fetchOpenAlexItems(ctx, paperLimit-len(items))
		items = append(items, openAlexItems...)
	}

	gitHubItems, gitHubErr := s.fetchGitHubItems(ctx, gitHubLimit)
	items = append(items, gitHubItems...)

	items = dedupeAndSortItems(items)
	if len(items) > limit {
		items = items[:limit]
	}
	if len(items) == 0 {
		if arxivErr != nil {
			return nil, arxivErr
		}
		if gitHubErr != nil {
			return nil, gitHubErr
		}
		return nil, fmt.Errorf("在线前沿资料为空")
	}

	return &frontierdto.LibraryResponse{
		Status:  "ready",
		Message: "已从 arXiv/OpenAlex 论文索引和 GitHub 公开仓库更新同步真实前沿资料；系统按 7 天缓存，下一次自动刷新后会更新列表。",
		Total:   len(items),
		Items:   items,
	}, nil
}

func (s *liveSource) fetchArxivItems(ctx context.Context, limit int) ([]frontierdto.LibraryItemView, error) {
	if limit <= 0 {
		return []frontierdto.LibraryItemView{}, nil
	}
	params := url.Values{}
	params.Set("search_query", `(cat:cs.AI OR cat:cs.CL OR cat:cs.LG OR cat:cs.IR) AND (all:"large language model" OR all:RAG OR all:Transformer OR all:"knowledge graph" OR all:agent OR all:"multimodal")`)
	params.Set("start", "0")
	params.Set("max_results", strconv.Itoa(limit))
	params.Set("sortBy", "submittedDate")
	params.Set("sortOrder", "descending")

	requestURL := s.arxivEndpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "course-recommend-agents/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("arXiv API 返回状态码 %d", resp.StatusCode)
	}

	var feed arxivFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	items := make([]frontierdto.LibraryItemView, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		item := arxivEntryToItem(entry)
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *liveSource) fetchGitHubItems(ctx context.Context, limit int) ([]frontierdto.LibraryItemView, error) {
	if limit <= 0 {
		return []frontierdto.LibraryItemView{}, nil
	}
	params := url.Values{}
	pushedSince := s.now().UTC().AddDate(0, -4, 0).Format("2006-01-02")
	params.Set("q", "topic:llm stars:>300 pushed:>="+pushedSince)
	params.Set("sort", "updated")
	params.Set("order", "desc")
	params.Set("per_page", strconv.Itoa(minInt(limit, 10)))

	requestURL := s.gitHubEndpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "course-recommend-agents/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub Search API 返回状态码 %d", resp.StatusCode)
	}

	var payload githubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	items := make([]frontierdto.LibraryItemView, 0, len(payload.Items))
	for _, repo := range payload.Items {
		item := gitHubRepoToItem(repo)
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *liveSource) fetchOpenAlexItems(ctx context.Context, limit int) ([]frontierdto.LibraryItemView, error) {
	if limit <= 0 {
		return []frontierdto.LibraryItemView{}, nil
	}
	now := s.now().UTC()
	params := url.Values{}
	params.Set("search", "large language model agent knowledge graph RAG transformer")
	params.Set("filter", "from_publication_date:"+now.AddDate(0, -6, 0).Format("2006-01-02")+",to_publication_date:"+now.Format("2006-01-02")+",type:article")
	params.Set("sort", "publication_date:desc")
	params.Set("per-page", strconv.Itoa(minInt(limit, 10)))
	params.Set("select", "id,doi,title,display_name,publication_year,publication_date,primary_location,authorships,abstract_inverted_index")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.openAlexEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "course-recommend-agents/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OpenAlex API 返回状态码 %d", resp.StatusCode)
	}

	var payload openAlexResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	items := make([]frontierdto.LibraryItemView, 0, len(payload.Results))
	for _, work := range payload.Results {
		item := openAlexWorkToItem(work)
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

type arxivFeed struct {
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID         string          `xml:"id"`
	Title      string          `xml:"title"`
	Summary    string          `xml:"summary"`
	Updated    string          `xml:"updated"`
	Published  string          `xml:"published"`
	Authors    []arxivAuthor   `xml:"author"`
	Categories []arxivCategory `xml:"category"`
}

type arxivAuthor struct {
	Name string `xml:"name"`
}

type arxivCategory struct {
	Term string `xml:"term,attr"`
}

type githubSearchResponse struct {
	Items []githubRepo `json:"items"`
}

type githubRepo struct {
	ID          int64    `json:"id"`
	FullName    string   `json:"full_name"`
	HTMLURL     string   `json:"html_url"`
	Description string   `json:"description"`
	UpdatedAt   string   `json:"updated_at"`
	PushedAt    string   `json:"pushed_at"`
	Language    string   `json:"language"`
	Stars       int      `json:"stargazers_count"`
	Topics      []string `json:"topics"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type openAlexResponse struct {
	Results []openAlexWork `json:"results"`
}

type openAlexWork struct {
	ID                    string           `json:"id"`
	DOI                   string           `json:"doi"`
	Title                 string           `json:"title"`
	DisplayName           string           `json:"display_name"`
	PublicationYear       int              `json:"publication_year"`
	PublicationDate       string           `json:"publication_date"`
	AbstractInvertedIndex map[string][]int `json:"abstract_inverted_index"`
	PrimaryLocation       openAlexLocation `json:"primary_location"`
	Authorships           []openAlexAuthor `json:"authorships"`
}

type openAlexLocation struct {
	LandingPageURL string         `json:"landing_page_url"`
	Source         openAlexSource `json:"source"`
}

type openAlexSource struct {
	DisplayName string `json:"display_name"`
}

type openAlexAuthor struct {
	Author openAlexAuthorName `json:"author"`
}

type openAlexAuthorName struct {
	DisplayName string `json:"display_name"`
}

func arxivEntryToItem(entry arxivEntry) frontierdto.LibraryItemView {
	title := cleanText(entry.Title)
	abstract := cleanText(entry.Summary)
	updatedAt := normalizeTimeString(entry.Updated)
	publishedAt := normalizeTimeString(entry.Published)
	year := yearFromTimeString(publishedAt)
	if year == 0 {
		year = yearFromTimeString(updatedAt)
	}
	url := strings.Replace(strings.TrimSpace(entry.ID), "http://", "https://", 1)
	paperID := strings.TrimPrefix(strings.TrimPrefix(url, "https://arxiv.org/abs/"), "http://arxiv.org/abs/")

	categories := make([]string, 0, len(entry.Categories))
	for _, category := range entry.Categories {
		if strings.TrimSpace(category.Term) != "" {
			categories = append(categories, category.Term)
		}
	}
	authors := make([]string, 0, minInt(len(entry.Authors), 4))
	for _, author := range entry.Authors {
		if len(authors) >= 4 {
			break
		}
		if name := strings.TrimSpace(author.Name); name != "" {
			authors = append(authors, name)
		}
	}
	topics := inferTopics(strings.Join([]string{title, abstract, strings.Join(categories, " ")}, " "))

	return frontierdto.LibraryItemView{
		PaperID:       paperID,
		Title:         title,
		Abstract:      abstract,
		Venue:         "arXiv",
		Year:          year,
		URL:           url,
		Source:        "arxiv",
		UpdatedAt:     updatedAt,
		PublishedAt:   publishedAt,
		Authors:       authors,
		Topics:        topics,
		KnowledgeRefs: inferKnowledgeRefs(topics),
	}
}

func gitHubRepoToItem(repo githubRepo) frontierdto.LibraryItemView {
	updatedAt := normalizeTimeString(firstNonEmpty(repo.PushedAt, repo.UpdatedAt))
	year := yearFromTimeString(updatedAt)
	title := strings.TrimSpace(repo.FullName)
	abstractParts := []string{cleanText(repo.Description)}
	meta := []string{}
	if strings.TrimSpace(repo.Language) != "" {
		meta = append(meta, repo.Language)
	}
	if repo.Stars > 0 {
		meta = append(meta, fmt.Sprintf("%d stars", repo.Stars))
	}
	if len(meta) > 0 {
		abstractParts = append(abstractParts, "公开仓库元数据："+strings.Join(meta, " / "))
	}
	abstract := strings.TrimSpace(strings.Join(nonEmptyStrings(abstractParts), "。"))
	if abstract == "" {
		abstract = "GitHub 公开仓库，按最近更新和主题标签筛选，用于追踪 AI 工具和工程实践方向。"
	}

	topics := inferTopics(strings.Join([]string{title, abstract, strings.Join(repo.Topics, " ")}, " "))
	if len(topics) == 0 {
		topics = []frontierdto.TopicView{{TopicID: "open-source-ai", Name: "开源 AI 工具", Reason: "来自 GitHub 最近更新的公开 AI 项目。"}}
	}

	return frontierdto.LibraryItemView{
		PaperID:       "github:" + strconv.FormatInt(repo.ID, 10),
		Title:         title,
		Abstract:      abstract,
		Venue:         "GitHub",
		Year:          year,
		URL:           strings.TrimSpace(repo.HTMLURL),
		Source:        "github",
		UpdatedAt:     updatedAt,
		Authors:       nonEmptyStrings([]string{repo.Owner.Login}),
		Topics:        topics,
		KnowledgeRefs: inferKnowledgeRefs(topics),
	}
}

func openAlexWorkToItem(work openAlexWork) frontierdto.LibraryItemView {
	title := cleanText(firstNonEmpty(work.Title, work.DisplayName))
	abstract := cleanText(reconstructOpenAlexAbstract(work.AbstractInvertedIndex))
	if abstract == "" {
		abstract = "OpenAlex 公开论文索引中的最新 AI 相关论文记录。"
	}
	publishedAt := ""
	if strings.TrimSpace(work.PublicationDate) != "" {
		publishedAt = strings.TrimSpace(work.PublicationDate) + "T00:00:00Z"
	}
	urlValue := sanitizeURL(firstNonEmpty(work.PrimaryLocation.LandingPageURL, work.DOI, work.ID))
	if strings.HasPrefix(strings.ToLower(urlValue), "https://doi.org/") || strings.HasPrefix(strings.ToLower(urlValue), "http://doi.org/") {
		urlValue = strings.Replace(urlValue, "http://", "https://", 1)
	}
	venue := firstNonEmpty(work.PrimaryLocation.Source.DisplayName, "OpenAlex")
	authors := make([]string, 0, minInt(len(work.Authorships), 4))
	for _, authorship := range work.Authorships {
		if len(authors) >= 4 {
			break
		}
		if name := strings.TrimSpace(authorship.Author.DisplayName); name != "" {
			authors = append(authors, name)
		}
	}
	topics := inferTopics(title + " " + abstract)
	return frontierdto.LibraryItemView{
		PaperID:       "openalex:" + strings.TrimPrefix(work.ID, "https://openalex.org/"),
		Title:         title,
		Abstract:      abstract,
		Venue:         venue,
		Year:          work.PublicationYear,
		URL:           urlValue,
		Source:        "openalex",
		UpdatedAt:     publishedAt,
		PublishedAt:   publishedAt,
		Authors:       authors,
		Topics:        topics,
		KnowledgeRefs: inferKnowledgeRefs(topics),
	}
}

type topicRule struct {
	id       string
	name     string
	reason   string
	keywords []string
}

var topicRules = []topicRule{
	{id: "rag", name: "检索增强生成", reason: "与 RAG、外部知识检索或知识密集型生成任务相关。", keywords: []string{"retrieval augmented", "retrieval-augmented", "rag", "retriever"}},
	{id: "agent", name: "Agent 与工具调用", reason: "涉及智能体规划、工具调用、多代理协作或自动化执行。", keywords: []string{"agent", "agents", "tool use", "tool-use", "multi-agent", "workflow"}},
	{id: "transformer", name: "Transformer", reason: "围绕 Transformer、注意力机制或大模型骨干架构展开。", keywords: []string{"transformer", "attention", "large language model", "llm"}},
	{id: "knowledge-graph", name: "知识图谱推理", reason: "涉及知识图谱、结构化知识、图推理或图检索。", keywords: []string{"knowledge graph", "graph reasoning", "graph retrieval", "kg"}},
	{id: "multimodal", name: "多模态学习", reason: "覆盖视觉-语言、多模态理解或生成模型。", keywords: []string{"multimodal", "multi-modal", "vision-language", "image", "video"}},
	{id: "safety", name: "模型安全与评测", reason: "关注模型安全、对齐、监控、红队测试或评测方法。", keywords: []string{"safety", "alignment", "jailbreak", "monitor", "evaluation", "benchmark"}},
	{id: "optimization", name: "优化与训练", reason: "与梯度优化、分布式训练或模型训练效率相关。", keywords: []string{"optimization", "gradient", "distributed learning", "training", "feedback"}},
	{id: "nlp", name: "自然语言处理", reason: "围绕文本理解、语言模型或自然语言任务展开。", keywords: []string{"language", "text", "nlp", "translation", "reasoning"}},
}

func inferTopics(text string) []frontierdto.TopicView {
	normalized := strings.ToLower(text)
	result := []frontierdto.TopicView{}
	for _, rule := range topicRules {
		for _, keyword := range rule.keywords {
			if strings.Contains(normalized, strings.ToLower(keyword)) {
				result = append(result, frontierdto.TopicView{TopicID: rule.id, Name: rule.name, Reason: rule.reason})
				break
			}
		}
	}
	if len(result) > 0 {
		return result
	}
	return []frontierdto.TopicView{{TopicID: "ai-frontier", Name: "AI 前沿研究", Reason: "来自公开前沿来源的最新 AI 相关资料。"}}
}

func inferKnowledgeRefs(topics []frontierdto.TopicView) []frontierdto.KnowledgeRefView {
	refs := []frontierdto.KnowledgeRefView{}
	for _, topic := range topics {
		switch topic.TopicID {
		case "rag":
			refs = append(refs, knowledgeRef("rag", "检索增强生成 RAG", "大模型", "该资料涉及外部检索与生成模型结合。"))
		case "agent":
			refs = append(refs, knowledgeRef("agent-tool", "Agent 与工具调用", "大模型", "该资料涉及智能体、工具调用或自动化流程。"))
		case "transformer":
			refs = append(refs, knowledgeRef("transformer", "Transformer 架构", "大模型", "该资料围绕 Transformer 或大模型架构展开。"))
		case "knowledge-graph":
			refs = append(refs, knowledgeRef("knowledge-graph", "知识图谱构建与查询", "知识图谱", "该资料与结构化知识组织和图谱推理相关。"))
		case "multimodal":
			refs = append(refs, knowledgeRef("cnn", "卷积神经网络 CNN", "深度学习", "多模态资料通常需要视觉表征基础。"))
		case "safety":
			refs = append(refs, knowledgeRef("model-eval", "模型评估与调参", "机器学习", "该资料涉及模型评测、安全监控或可靠性分析。"))
		case "optimization":
			refs = append(refs, knowledgeRef("gradient-descent", "梯度下降", "机器学习", "该资料涉及训练优化和参数更新机制。"))
		case "nlp":
			refs = append(refs, knowledgeRef("sentiment", "文本情感分析", "行业实践", "该资料属于自然语言处理相关延伸。"))
		}
	}
	return uniqueKnowledgeRefs(refs, 4)
}

func knowledgeRef(id string, name string, course string, reason string) frontierdto.KnowledgeRefView {
	return frontierdto.KnowledgeRefView{KnowledgeID: id, Name: name, Course: course, Reason: reason}
}

func filterItemsByKnowledge(items []frontierdto.LibraryItemView, knowledge string) []frontierdto.LibraryItemView {
	terms := knowledgeSearchTerms(knowledge)
	if len(terms) == 0 {
		return []frontierdto.LibraryItemView{}
	}
	result := []frontierdto.LibraryItemView{}
	for _, item := range items {
		text := strings.ToLower(strings.Join([]string{
			item.Title,
			item.Abstract,
			item.Venue,
			strings.Join(topicNames(item.Topics), " "),
			strings.Join(knowledgeNames(item.KnowledgeRefs), " "),
		}, " "))
		for _, term := range terms {
			if strings.Contains(text, strings.ToLower(term)) {
				result = append(result, item)
				break
			}
		}
	}
	return result
}

func knowledgeSearchTerms(knowledge string) []string {
	normalized := strings.ToLower(strings.TrimSpace(knowledge))
	terms := nonEmptyStrings([]string{knowledge, strings.ReplaceAll(knowledge, "-", " ")})
	switch {
	case strings.Contains(normalized, "rag") || strings.Contains(normalized, "检索"):
		terms = append(terms, "rag", "retrieval augmented", "检索增强生成")
	case strings.Contains(normalized, "transformer") || strings.Contains(normalized, "attention") || strings.Contains(normalized, "注意力"):
		terms = append(terms, "transformer", "attention", "large language model")
	case strings.Contains(normalized, "agent") || strings.Contains(normalized, "智能体"):
		terms = append(terms, "agent", "tool use", "multi-agent")
	case strings.Contains(normalized, "knowledge") || strings.Contains(normalized, "图谱"):
		terms = append(terms, "knowledge graph", "graph reasoning")
	case strings.Contains(normalized, "gradient") || strings.Contains(normalized, "optimization") || strings.Contains(normalized, "梯度") || strings.Contains(normalized, "优化"):
		terms = append(terms, "gradient", "optimization", "training")
	case strings.Contains(normalized, "cnn") || strings.Contains(normalized, "视觉") || strings.Contains(normalized, "图像"):
		terms = append(terms, "image", "vision", "multimodal")
	case strings.Contains(normalized, "nlp") || strings.Contains(normalized, "自然语言") || strings.Contains(normalized, "文本"):
		terms = append(terms, "language", "text", "nlp")
	}
	return uniqueStrings(terms)
}

func dedupeAndSortItems(items []frontierdto.LibraryItemView) []frontierdto.LibraryItemView {
	seen := map[string]struct{}{}
	result := make([]frontierdto.LibraryItemView, 0, len(items))
	for _, item := range items {
		key := strings.ToLower(firstNonEmpty(item.URL, item.Title))
		if strings.TrimSpace(key) == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return parseTimeForSort(result[i].UpdatedAt).After(parseTimeForSort(result[j].UpdatedAt))
	})
	return result
}

func limitLibraryResponse(response *frontierdto.LibraryResponse, limit int) *frontierdto.LibraryResponse {
	if response == nil {
		return nil
	}
	next := cloneLibraryResponse(response)
	if limit > 0 && len(next.Items) > limit {
		next.Items = next.Items[:limit]
	}
	next.Total = len(next.Items)
	return next
}

func cloneLibraryResponse(response *frontierdto.LibraryResponse) *frontierdto.LibraryResponse {
	if response == nil {
		return nil
	}
	next := *response
	next.Items = make([]frontierdto.LibraryItemView, len(response.Items))
	copy(next.Items, response.Items)
	return &next
}

func uniqueTopics(items []frontierdto.LibraryItemView, limit int) []frontierdto.TopicView {
	seen := map[string]struct{}{}
	result := []frontierdto.TopicView{}
	for _, item := range items {
		for _, topic := range item.Topics {
			key := strings.TrimSpace(firstNonEmpty(topic.TopicID, topic.Name))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, topic)
			if limit > 0 && len(result) >= limit {
				return result
			}
		}
	}
	return result
}

func uniqueKnowledgeRefs(refs []frontierdto.KnowledgeRefView, limit int) []frontierdto.KnowledgeRefView {
	seen := map[string]struct{}{}
	result := []frontierdto.KnowledgeRefView{}
	for _, ref := range refs {
		key := strings.TrimSpace(firstNonEmpty(ref.KnowledgeID, ref.Name))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, ref)
		if limit > 0 && len(result) >= limit {
			return result
		}
	}
	return result
}

func notReadyLiveLibraryResponse(err error) *frontierdto.LibraryResponse {
	message := "暂时无法连接 arXiv 或 GitHub 公开来源，请稍后刷新。"
	if err != nil {
		message = message + " 原因：" + err.Error()
	}
	return &frontierdto.LibraryResponse{
		Status:  "not_ready",
		Message: message,
		Items:   []frontierdto.LibraryItemView{},
	}
}

func topicNames(topics []frontierdto.TopicView) []string {
	result := make([]string, 0, len(topics))
	for _, topic := range topics {
		result = append(result, topic.Name)
	}
	return result
}

func knowledgeNames(refs []frontierdto.KnowledgeRefView) []string {
	result := make([]string, 0, len(refs))
	for _, ref := range refs {
		result = append(result, ref.Name)
	}
	return result
}

func normalizeBriefingID(knowledgeID string, knowledgeName string) string {
	if strings.TrimSpace(knowledgeID) != "" {
		return strings.TrimSpace(knowledgeID)
	}
	return strings.TrimSpace(knowledgeName)
}

func normalizeBriefingKnowledge(knowledgeID string, knowledgeName string) string {
	if strings.TrimSpace(knowledgeName) != "" {
		return strings.TrimSpace(knowledgeName)
	}
	return strings.TrimSpace(knowledgeID)
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeTimeString(value string) string {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	return parsed.UTC().Format(time.RFC3339)
}

func yearFromTimeString(value string) int {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed.Year()
}

func parseTimeForSort(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizeURL(value string) string {
	return strings.Trim(strings.TrimSpace(value), "<>\"'")
}

func reconstructOpenAlexAbstract(index map[string][]int) string {
	if len(index) == 0 {
		return ""
	}
	type tokenAt struct {
		token string
		pos   int
	}
	tokens := []tokenAt{}
	for token, positions := range index {
		for _, pos := range positions {
			tokens = append(tokens, tokenAt{token: token, pos: pos})
		}
	}
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].pos < tokens[j].pos
	})
	words := make([]string, 0, len(tokens))
	for _, item := range tokens {
		words = append(words, item.token)
	}
	return strings.Join(words, " ")
}

func nonEmptyStrings(values []string) []string {
	result := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range nonEmptyStrings(values) {
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
