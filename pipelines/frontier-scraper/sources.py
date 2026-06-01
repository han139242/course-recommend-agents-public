from __future__ import annotations

from dataclasses import dataclass, field

from arxiv_fetcher import AI_CATEGORIES, DEFAULT_AI_QUERIES


@dataclass(frozen=True)
class GitHubFeedSource:
    name: str
    url: str
    kind: str
    knowledge_hints: list[str] = field(default_factory=list)
    topic_hints: list[str] = field(default_factory=list)


DEFAULT_ARXIV_CATEGORIES: tuple[str, ...] = tuple(AI_CATEGORIES.keys())

DEFAULT_SOURCE_POLICY: dict[str, object] = {
    "arxiv_categories": DEFAULT_ARXIV_CATEGORIES,
    "arxiv_queries": DEFAULT_AI_QUERIES,
    "paper_window_days": 14,
    "max_per_category": 5,
    "github_release_limit": 10,
}

# 这些 GitHub 源不是论文数据库本身，而是持续更新的前沿线索源。
# 同步器默认仅把它们作为“来源观察列表”和可选 release 事件，不替代 arXiv / 论文源。
DEFAULT_GITHUB_FEEDS: tuple[GitHubFeedSource, ...] = (
    GitHubFeedSource(
        name="Papers with Code",
        url="https://github.com/paperswithcode/paperswithcode-data",
        kind="repository",
        knowledge_hints=["机器学习", "AI 评测"],
        topic_hints=["开源论文与代码索引"],
    ),
    GitHubFeedSource(
        name="Hugging Face Papers",
        url="https://github.com/huggingface/papers",
        kind="repository",
        knowledge_hints=["大语言模型", "多模态学习"],
        topic_hints=["社区论文趋势"],
    ),
    GitHubFeedSource(
        name="Awesome Deep Learning Papers",
        url="https://github.com/terryum/awesome-deep-learning-papers",
        kind="repository",
        knowledge_hints=["深度学习"],
        topic_hints=["经典与前沿论文清单"],
    ),
)


def github_repo_api_url(repo_url: str, endpoint: str = "releases") -> str:
    repo_url = repo_url.rstrip("/")
    marker = "github.com/"
    if marker not in repo_url:
        raise ValueError(f"not a GitHub repository url: {repo_url}")
    owner_repo = repo_url.split(marker, 1)[1].strip("/")
    parts = owner_repo.split("/")
    if len(parts) < 2:
        raise ValueError(f"not a GitHub repository url: {repo_url}")
    owner, repo = parts[0], parts[1]
    endpoint = endpoint.strip("/")
    return f"https://api.github.com/repos/{owner}/{repo}/{endpoint}"
