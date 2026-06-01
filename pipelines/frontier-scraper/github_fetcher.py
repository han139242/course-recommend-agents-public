from __future__ import annotations

import json
import urllib.request
from datetime import UTC, datetime
from typing import Iterable

from arxiv_fetcher import USER_AGENT, dedupe, normalize_space
from sources import GitHubFeedSource, github_repo_api_url


def _parse_iso(value: str) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(UTC)
    except ValueError:
        return None


def http_get_json(url: str, timeout: int = 30, token: str = ""):
    headers = {
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "User-Agent": USER_AGENT,
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(request, timeout=timeout) as response:
        return json.loads(response.read().decode("utf-8"))


def fetch_repository_releases(
    source: GitHubFeedSource,
    limit: int = 10,
    since_days: int | None = None,
    token: str = "",
) -> list[dict]:
    api_url = github_repo_api_url(source.url, "releases")
    payload = http_get_json(f"{api_url}?per_page={max(1, min(limit, 100))}", token=token)
    if not isinstance(payload, list):
        return []

    cutoff = None
    if since_days and since_days > 0:
        cutoff = datetime.now(UTC).timestamp() - since_days * 24 * 3600

    papers: list[dict] = []
    for item in payload[:limit]:
        if not isinstance(item, dict):
            continue
        published_at = normalize_space(item.get("published_at") or item.get("created_at") or "")
        parsed = _parse_iso(published_at)
        if cutoff and parsed and parsed.timestamp() < cutoff:
            continue
        title = normalize_space(item.get("name") or item.get("tag_name") or "")
        body = normalize_space(item.get("body") or "")
        url = normalize_space(item.get("html_url") or source.url)
        if not title:
            continue
        papers.append(
            {
                "title": f"{source.name}: {title}",
                "abstract": body[:2000] or f"{source.name} 发布了新的开源前沿信息，来源仓库为 {source.url}。",
                "keywords": dedupe([source.name, "GitHub", "Open Source AI"]),
                "year": parsed.year if parsed else None,
                "venue": "GitHub",
                "venueType": "open-source",
                "venueRank": "",
                "url": url,
                "authors": [],
                "updatedAt": published_at,
                "knowledgeHints": source.knowledge_hints,
                "topicHints": source.topic_hints,
                "source": "github",
                "sourceId": normalize_space(item.get("id", "")) or url,
                "sourceName": source.name,
                "category": "github.release",
            }
        )
    return papers


def fetch_github_releases(
    sources: Iterable[GitHubFeedSource],
    limit: int = 10,
    since_days: int | None = None,
    token: str = "",
) -> list[dict]:
    papers: list[dict] = []
    for source in sources:
        try:
            papers.extend(
                fetch_repository_releases(
                    source=source,
                    limit=limit,
                    since_days=since_days,
                    token=token,
                )
            )
        except Exception as exc:
            papers.append(
                {
                    "title": f"{source.name}: 同步失败",
                    "abstract": f"GitHub 源 {source.url} 暂时无法读取 releases：{exc}",
                    "keywords": ["GitHub", "同步状态"],
                    "year": datetime.now(UTC).year,
                    "venue": "GitHub",
                    "venueType": "open-source",
                    "url": source.url,
                    "authors": [],
                    "updatedAt": datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
                    "knowledgeHints": source.knowledge_hints,
                    "topicHints": ["开源前沿信息同步"],
                    "source": "github",
                    "sourceId": source.url,
                    "sourceName": source.name,
                    "category": "github.release.error",
                }
            )
    return papers
