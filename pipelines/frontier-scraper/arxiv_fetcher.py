from __future__ import annotations

import re
import time
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from dataclasses import dataclass, field
from datetime import UTC, date, datetime, timedelta
from email.utils import parsedate_to_datetime
from typing import Iterable

ARXIV_API_BASE = "https://export.arxiv.org/api/query"
ARXIV_ABS_BASE = "https://arxiv.org/abs/"
ARXIV_REQUEST_DELAY_SECONDS = 3.1
USER_AGENT = "course-recommend-frontier-sync/2.0 (educational knowledge graph)"

AI_CATEGORIES: dict[str, str] = {
    "cs.AI": "人工智能",
    "cs.LG": "机器学习",
    "cs.CL": "自然语言处理",
    "cs.CV": "计算机视觉",
    "cs.NE": "神经网络与进化计算",
    "cs.IR": "信息检索",
    "cs.MA": "多智能体系统",
    "cs.RO": "机器人学",
    "stat.ML": "统计机器学习",
}

DEFAULT_AI_QUERIES: tuple[str, ...] = (
    '"large language model"',
    '"retrieval augmented generation"',
    '"agent"',
    '"multimodal"',
    '"diffusion model"',
    '"reinforcement learning"',
    '"transformer"',
    '"knowledge graph"',
    '"AI safety"',
)

KEYWORD_RULES: tuple[tuple[tuple[str, ...], str, str], ...] = (
    (("large language model", "llm", "foundation model"), "Large Language Model", "大语言模型"),
    (("retrieval augmented", "rag", "retrieval-augmented"), "Retrieval-Augmented Generation", "检索增强生成"),
    (("transformer",), "Transformer", "Transformer"),
    (("attention", "self-attention"), "Attention Mechanism", "注意力机制"),
    (("diffusion", "score-based generative"), "Diffusion Model", "扩散模型"),
    (("reinforcement learning", "policy gradient", "q-learning"), "Reinforcement Learning", "强化学习"),
    (("multi-agent", "multiagent"), "Multi-Agent System", "多智能体系统"),
    (("agent", "tool use", "function calling"), "AI Agent", "智能体"),
    (("graph neural", "gnn"), "Graph Neural Network", "图神经网络"),
    (("knowledge graph", "knowledge base"), "Knowledge Graph", "知识图谱"),
    (("computer vision", "vision transformer", "image"), "Computer Vision", "计算机视觉"),
    (("natural language", "nlp", "language model"), "Natural Language Processing", "自然语言处理"),
    (("multimodal", "multi-modal", "vision-language"), "Multimodal Learning", "多模态学习"),
    (("federated",), "Federated Learning", "联邦学习"),
    (("privacy", "differential privacy"), "Privacy-Preserving Machine Learning", "隐私保护机器学习"),
    (("adversarial", "robustness", "jailbreak", "red teaming"), "AI Safety", "AI 安全"),
    (("alignment", "rlhf", "constitutional ai"), "AI Alignment", "AI 对齐"),
    (("quantization", "distillation", "pruning"), "Model Compression", "模型压缩"),
    (("serving", "inference", "latency", "throughput"), "Efficient Inference", "高效推理"),
    (("robot", "embodied", "manipulation", "navigation"), "Embodied AI", "具身智能"),
    (("causal", "counterfactual"), "Causal Inference", "因果推断"),
    (("time series", "forecasting"), "Time Series Forecasting", "时间序列预测"),
    (("automl", "neural architecture search"), "AutoML", "自动机器学习"),
    (("benchmark", "evaluation", "eval"), "AI Evaluation", "AI 评测"),
)


@dataclass(frozen=True)
class PaperMetadata:
    title: str
    abstract: str
    keywords: list[str] = field(default_factory=list)
    year: int | None = None
    venue: str = "arXiv"
    venue_type: str = "preprint"
    venue_rank: str = ""
    url: str = ""
    authors: list[dict[str, str]] = field(default_factory=list)
    updated_at: str = ""
    knowledge_hints: list[str] = field(default_factory=list)
    topic_hints: list[str] = field(default_factory=list)
    source: str = "arxiv"
    source_id: str = ""
    source_name: str = "arXiv"
    category: str = ""


def normalize_space(text: str) -> str:
    return re.sub(r"\s+", " ", text or "").strip()


def arxiv_date_window(days: int | None) -> str:
    if not days or days <= 0:
        return ""
    end = datetime.now(UTC)
    start = end - timedelta(days=days)
    return f"submittedDate:[{start:%Y%m%d%H%M} TO {end:%Y%m%d%H%M}]"


def build_search_query(category: str, query: str = "", since_days: int | None = None) -> str:
    parts = [f"cat:{category}"]
    if query.strip():
        parts.append(f"all:{query.strip()}")
    window = arxiv_date_window(since_days)
    if window:
        parts.append(window)
    return " AND ".join(parts)


def build_arxiv_url(
    category: str,
    query: str = "",
    max_results: int = 10,
    start: int = 0,
    since_days: int | None = None,
) -> str:
    params = {
        "search_query": build_search_query(category=category, query=query, since_days=since_days),
        "start": str(max(0, start)),
        "max_results": str(max(1, min(max_results, 100))),
        "sortBy": "submittedDate",
        "sortOrder": "descending",
    }
    return f"{ARXIV_API_BASE}?{urllib.parse.urlencode(params)}"


def _atom(tag: str) -> str:
    return f"{{http://www.w3.org/2005/Atom}}{tag}"


def _arxiv(tag: str) -> str:
    return f"{{http://arxiv.org/schemas/atom}}{tag}"


def _text(parent: ET.Element | None, tag: str) -> str:
    child = parent.find(_atom(tag)) if parent is not None else None
    if child is None or child.text is None:
        return ""
    return normalize_space(child.text)


def _parse_iso(value: str) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(UTC)
    except ValueError:
        return None


def _parse_year(value: str) -> int | None:
    parsed = _parse_iso(value)
    return parsed.year if parsed else None


def _parse_retry_after(value: str) -> float | None:
    if not value:
        return None
    try:
        return max(0.0, float(value))
    except ValueError:
        pass
    try:
        parsed = parsedate_to_datetime(value)
    except (TypeError, ValueError):
        return None
    return max(0.0, (parsed.astimezone(UTC) - datetime.now(UTC)).total_seconds())


def extract_arxiv_id(entry: ET.Element) -> str:
    raw_id = _text(entry, "id")
    if "/abs/" not in raw_id:
        return ""
    arxiv_id = raw_id.rsplit("/abs/", 1)[-1]
    return re.sub(r"v\d+$", "", arxiv_id).strip()


def extract_primary_category(entry: ET.Element) -> str:
    primary = entry.find(_arxiv("primary_category"))
    if primary is not None:
        term = normalize_space(primary.attrib.get("term", ""))
        if term:
            return term
    category = entry.find(_atom("category"))
    return normalize_space(category.attrib.get("term", "")) if category is not None else ""


def extract_categories(entry: ET.Element) -> list[str]:
    categories: list[str] = []
    for item in entry.findall(_atom("category")):
        term = normalize_space(item.attrib.get("term", ""))
        if term and term not in categories:
            categories.append(term)
    return categories


def extract_keywords(title: str, abstract: str, category: str = "") -> tuple[list[str], list[str], list[str]]:
    haystack = f"{title} {abstract}".lower()
    keywords: list[str] = []
    knowledge_hints: list[str] = []
    topic_hints: list[str] = []

    for patterns, keyword, hint in KEYWORD_RULES:
        if any(pattern in haystack for pattern in patterns):
            keywords.append(keyword)
            knowledge_hints.append(hint)
            topic_hints.append(hint)

    category_label = AI_CATEGORIES.get(category)
    if category_label:
        knowledge_hints.append(category_label)
        topic_hints.append(category_label)

    return dedupe(keywords)[:8], dedupe(knowledge_hints)[:6], dedupe(topic_hints)[:6]


def parse_arxiv_feed(xml_text: str, requested_category: str = "") -> list[PaperMetadata]:
    root = ET.fromstring(xml_text)
    papers: list[PaperMetadata] = []

    for entry in root.findall(_atom("entry")):
        title = normalize_space(_text(entry, "title"))
        abstract = normalize_space(_text(entry, "summary"))
        if not title or not abstract:
            continue

        arxiv_id = extract_arxiv_id(entry)
        published = _text(entry, "published")
        updated = _text(entry, "updated")
        category = extract_primary_category(entry) or requested_category
        all_categories = extract_categories(entry)
        keywords, knowledge_hints, topic_hints = extract_keywords(title, abstract, category)
        for extra_category in all_categories:
            label = AI_CATEGORIES.get(extra_category)
            if label:
                knowledge_hints.append(label)

        authors: list[dict[str, str]] = []
        for author_el in entry.findall(_atom("author")):
            name = _text(author_el, "name")
            if name:
                authors.append({"name": name, "affiliation": ""})

        papers.append(
            PaperMetadata(
                title=title,
                abstract=abstract,
                keywords=keywords,
                year=_parse_year(published),
                venue="arXiv",
                venue_type="preprint",
                venue_rank="",
                url=f"{ARXIV_ABS_BASE}{arxiv_id}" if arxiv_id else _text(entry, "id"),
                authors=authors,
                updated_at=updated or published,
                knowledge_hints=dedupe(knowledge_hints)[:8],
                topic_hints=dedupe(topic_hints)[:8],
                source="arxiv",
                source_id=arxiv_id,
                source_name="arXiv",
                category=category,
            )
        )

    return papers


def http_get_text(url: str, timeout: int = 30, user_agent: str = USER_AGENT, retries: int = 2) -> str:
    for attempt in range(retries + 1):
        request = urllib.request.Request(url, headers={"User-Agent": user_agent})
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                return response.read().decode("utf-8")
        except urllib.error.HTTPError as exc:
            retry_after = _parse_retry_after(exc.headers.get("Retry-After", ""))
            if exc.code in {429, 500, 502, 503, 504} and attempt < retries:
                time.sleep(retry_after or 2.0 * (attempt + 1))
                continue
            raise
        except urllib.error.URLError:
            if attempt < retries:
                time.sleep(2.0 * (attempt + 1))
                continue
            raise
    raise RuntimeError(f"request failed: {url}")


def fetch_arxiv_papers(
    category: str,
    query: str = "",
    max_results: int = 10,
    start: int = 0,
    since_days: int | None = None,
    timeout: int = 30,
) -> list[PaperMetadata]:
    url = build_arxiv_url(
        category=category,
        query=query,
        max_results=max_results,
        start=start,
        since_days=since_days,
    )
    xml_text = http_get_text(url, timeout=timeout)
    return parse_arxiv_feed(xml_text, requested_category=category)


def fetch_all_categories(
    categories: Iterable[str] | None = None,
    max_per_category: int = 5,
    query: str = "",
    since_days: int | None = None,
    request_delay: float = ARXIV_REQUEST_DELAY_SECONDS,
) -> list[PaperMetadata]:
    categories = list(categories or AI_CATEGORIES.keys())
    papers: list[PaperMetadata] = []
    seen: set[str] = set()

    for index, category in enumerate(categories):
        fetched = fetch_arxiv_papers(
            category=category,
            query=query,
            max_results=max_per_category,
            since_days=since_days,
        )
        for paper in fetched:
            key = dedupe_key(paper)
            if key in seen:
                continue
            seen.add(key)
            papers.append(paper)
        if index < len(categories) - 1:
            time.sleep(max(0.0, request_delay))

    return papers


def dedupe_key(paper: PaperMetadata | dict) -> str:
    if isinstance(paper, PaperMetadata):
        source_id = paper.source_id
        url = paper.url
        title = paper.title
    else:
        source_id = str(paper.get("sourceId", "") or paper.get("source_id", ""))
        url = str(paper.get("url", ""))
        title = str(paper.get("title", ""))
    if source_id:
        return f"id:{source_id.lower()}"
    if url:
        return f"url:{url.lower().rstrip('/')}"
    return f"title:{normalize_space(title).lower()}"


def dedupe(values: Iterable[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for value in values:
        item = normalize_space(value)
        if not item:
            continue
        key = item.casefold()
        if key in seen:
            continue
        seen.add(key)
        result.append(item)
    return result


def papers_to_dicts(papers: Iterable[PaperMetadata]) -> list[dict]:
    result: list[dict] = []
    for paper in papers:
        result.append(
            {
                "title": paper.title,
                "abstract": paper.abstract,
                "keywords": paper.keywords,
                "year": paper.year,
                "venue": paper.venue,
                "venueType": paper.venue_type,
                "venueRank": paper.venue_rank,
                "url": paper.url,
                "authors": paper.authors,
                "updatedAt": paper.updated_at,
                "knowledgeHints": paper.knowledge_hints,
                "topicHints": paper.topic_hints,
                "source": paper.source,
                "sourceId": paper.source_id,
                "sourceName": paper.source_name,
                "category": paper.category,
            }
        )
    return result


def today_utc() -> date:
    return datetime.now(UTC).date()
