from __future__ import annotations

import json
import urllib.parse
import urllib.request
from datetime import UTC, datetime

from arxiv_fetcher import KEYWORD_RULES, USER_AGENT, dedupe, normalize_space

HF_DAILY_PAPERS_API = "https://huggingface.co/api/daily_papers"


def _parse_iso(value: str) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(UTC)
    except ValueError:
        return None


def _extract_paper_items(payload) -> list[dict]:
    if isinstance(payload, list):
        return [item for item in payload if isinstance(item, dict)]
    if isinstance(payload, dict):
        for key in ("papers", "items", "data"):
            value = payload.get(key)
            if isinstance(value, list):
                return [item for item in value if isinstance(item, dict)]
    return []


def _collect_text_fields(item: dict) -> tuple[str, str, str]:
    paper = item.get("paper") if isinstance(item.get("paper"), dict) else {}
    title = normalize_space(item.get("title") or paper.get("title") or "")
    abstract = normalize_space(item.get("summary") or item.get("abstract") or paper.get("summary") or paper.get("abstract") or "")
    paper_id = normalize_space(item.get("paperId") or item.get("id") or paper.get("id") or paper.get("paperId") or "")
    return title, abstract, paper_id


def _derive_hints(title: str, abstract: str) -> tuple[list[str], list[str]]:
    haystack = f"{title} {abstract}".lower()
    keywords: list[str] = []
    hints: list[str] = []
    for patterns, keyword, hint in KEYWORD_RULES:
        if any(pattern in haystack for pattern in patterns):
            keywords.append(keyword)
            hints.append(hint)
    return dedupe(keywords)[:8], dedupe(hints)[:8]


def fetch_huggingface_daily_papers(limit: int = 20, timeout: int = 30) -> list[dict]:
    query = urllib.parse.urlencode({"limit": max(1, min(limit, 100))})
    request = urllib.request.Request(f"{HF_DAILY_PAPERS_API}?{query}", headers={"User-Agent": USER_AGENT})
    with urllib.request.urlopen(request, timeout=timeout) as response:
        payload = json.loads(response.read().decode("utf-8"))

    papers: list[dict] = []
    for item in _extract_paper_items(payload)[:limit]:
        title, abstract, paper_id = _collect_text_fields(item)
        if not title:
            continue
        published_at = normalize_space(
            item.get("publishedAt")
            or item.get("published_at")
            or item.get("createdAt")
            or item.get("created_at")
            or ""
        )
        parsed = _parse_iso(published_at)
        keywords, hints = _derive_hints(title, abstract)
        url = normalize_space(item.get("url") or item.get("paperUrl") or "")
        if not url and paper_id:
            url = f"https://huggingface.co/papers/{paper_id}"
        papers.append(
            {
                "title": title,
                "abstract": abstract or f"Hugging Face Daily Papers 收录的 AI 前沿论文：{title}",
                "keywords": keywords or ["Hugging Face Daily Papers"],
                "year": parsed.year if parsed else datetime.now(UTC).year,
                "venue": "Hugging Face Daily Papers",
                "venueType": "community-paper-feed",
                "venueRank": "",
                "url": url,
                "authors": [],
                "updatedAt": published_at or datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
                "knowledgeHints": hints or ["人工智能"],
                "topicHints": hints or ["AI 前沿论文"],
                "source": "huggingface_daily_papers",
                "sourceId": paper_id or url or title,
                "sourceName": "Hugging Face Daily Papers",
                "category": "hf.daily_papers",
            }
        )
    return papers
