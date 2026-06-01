from __future__ import annotations

import json
import os
import time
from pathlib import Path
from typing import Any

from dotenv import load_dotenv
from openai import OpenAI

from knowledge_catalog import (
    KnowledgeCandidate,
    candidate_name_set,
    candidates_to_prompt,
    normalize_key,
    select_knowledge_candidates,
)

for parent in [Path.cwd(), *Path.cwd().parents]:
    env_path = parent / ".env"
    if env_path.exists():
        load_dotenv(env_path)
        break

DEFAULT_MODEL = os.getenv("DEEPSEEK_MODEL", "deepseek-chat")

SYSTEM_PROMPT = """你是人工智能课程知识图谱专家。请把论文标题和摘要转成可落库的结构化前沿元数据。
只返回 JSON 对象，不要添加解释文字。字段必须包含：
{
  "keywords_cn": ["中文关键词"],
  "keywords_en": ["English keyword"],
  "knowledge_hints": ["课程知识点名称"],
  "topic_hints": ["前沿研究方向"],
  "summary_cn": "80 到 140 字中文摘要"
}
要求：
1. knowledge_hints 优先使用人工智能课程常见概念，例如 Transformer、注意力机制、强化学习、知识图谱、检索增强生成、图神经网络、多模态学习、AI 安全。
2. topic_hints 写研究方向，不要和 knowledge_hints 完全重复。
3. 不编造作者、年份、链接和会议。
4. 摘要只基于给定标题和摘要。"""

MAPPING_PROMPT = """你是人工智能专业课程知识图谱的审稿人。请在给定候选知识点中，选择与论文最相关的 2 到 6 个知识点。
只返回 JSON 对象，不要添加解释文字。字段必须包含：
{
  "knowledge_hints": ["必须来自候选知识点名称"],
  "topic_hints": ["前沿研究方向"],
  "mapping_reasons": [{"knowledge": "候选知识点名称", "reason": "一句话说明关联"}],
  "summary_cn": "80 到 140 字中文摘要"
}
规则：
1. knowledge_hints 必须优先从候选知识点原名中选择，不能随意创造课程知识点。
2. 如果候选中没有完全匹配项，可以选择更上位的课程概念。
3. topic_hints 写研究方向，允许不是课程知识点。
4. 不编造论文事实。"""


def _get_client() -> OpenAI:
    api_key = os.getenv("DEEPSEEK_API_KEY") or os.getenv("OPENAI_API_KEY")
    base_url = os.getenv("DEEPSEEK_BASE_URL") or os.getenv("OPENAI_BASE_URL") or "https://api.deepseek.com"
    if not api_key:
        raise RuntimeError("未设置 DEEPSEEK_API_KEY 或 OPENAI_API_KEY，无法启用 AI 摘要。")
    return OpenAI(api_key=api_key, base_url=base_url)


def _ensure_list(value: Any) -> list[str]:
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    if isinstance(value, str) and value.strip():
        return [value.strip()]
    return []


def _empty_result() -> dict[str, Any]:
    return {
        "keywords_cn": [],
        "keywords_en": [],
        "knowledge_hints": [],
        "topic_hints": [],
        "summary_cn": "",
    }


def enrich_paper(
    title: str,
    abstract: str,
    category: str = "",
    model: str | None = None,
    temperature: float = 0.2,
) -> dict[str, Any]:
    client = _get_client()
    model_name = model or DEFAULT_MODEL
    prompt = f"""论文标题：{title}
论文摘要：{abstract[:1800]}
来源分类：{category or "AI"}

请生成 keywords_cn、keywords_en、knowledge_hints、topic_hints 和 summary_cn。"""

    for attempt in range(3):
        try:
            response = client.chat.completions.create(
                model=model_name,
                messages=[
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": prompt},
                ],
                temperature=temperature,
                max_tokens=900,
                response_format={"type": "json_object"},
            )
            content = response.choices[0].message.content or "{}"
            raw = json.loads(content)
            return {
                "keywords_cn": _ensure_list(raw.get("keywords_cn")),
                "keywords_en": _ensure_list(raw.get("keywords_en")),
                "knowledge_hints": _ensure_list(raw.get("knowledge_hints")),
                "topic_hints": _ensure_list(raw.get("topic_hints")),
                "summary_cn": str(raw.get("summary_cn", "")).strip(),
            }
        except (json.JSONDecodeError, Exception) as exc:
            if attempt == 2:
                print(f"[AI] 摘要失败：{title[:80]} - {exc}")
                return _empty_result()
            time.sleep(1.5 * (attempt + 1))

    return _empty_result()


def map_paper_to_knowledge(
    paper: dict,
    candidates: list[KnowledgeCandidate] | None = None,
    model: str | None = None,
    temperature: float = 0.1,
) -> dict[str, Any]:
    selected_candidates = candidates or select_knowledge_candidates(
        title=str(paper.get("title", "")),
        abstract=str(paper.get("abstract", "")),
        keywords=paper.get("keywords") or [],
        seed_hints=paper.get("knowledgeHints") or [],
    )
    if not selected_candidates:
        return _empty_result() | {"mapping_reasons": []}

    client = _get_client()
    model_name = model or DEFAULT_MODEL
    allowed = candidate_name_set(selected_candidates)
    prompt = f"""论文标题：{paper.get('title', '')}
论文摘要：{str(paper.get('abstract', ''))[:1800]}
已有规则提示：{', '.join(paper.get('knowledgeHints') or [])}

候选课程知识点：
{candidates_to_prompt(selected_candidates)}

请只从候选知识点中选择 knowledge_hints，并给出 mapping_reasons。"""

    for attempt in range(3):
        try:
            response = client.chat.completions.create(
                model=model_name,
                messages=[
                    {"role": "system", "content": MAPPING_PROMPT},
                    {"role": "user", "content": prompt},
                ],
                temperature=temperature,
                max_tokens=1100,
                response_format={"type": "json_object"},
            )
            raw = json.loads(response.choices[0].message.content or "{}")
            hints = [
                item
                for item in _ensure_list(raw.get("knowledge_hints"))
                if normalize_key(item) in allowed
            ]
            reasons = []
            for item in raw.get("mapping_reasons", []) if isinstance(raw.get("mapping_reasons"), list) else []:
                if not isinstance(item, dict):
                    continue
                knowledge = str(item.get("knowledge", "")).strip()
                if normalize_key(knowledge) not in allowed:
                    continue
                reasons.append(
                    {
                        "knowledge": knowledge,
                        "reason": str(item.get("reason", "")).strip(),
                    }
                )
            return {
                "keywords_cn": [],
                "keywords_en": [],
                "knowledge_hints": hints,
                "topic_hints": _ensure_list(raw.get("topic_hints")),
                "summary_cn": str(raw.get("summary_cn", "")).strip(),
                "mapping_reasons": reasons,
            }
        except (json.JSONDecodeError, Exception) as exc:
            if attempt == 2:
                print(f"[AI] 知识点映射失败：{str(paper.get('title', ''))[:80]} - {exc}")
                return _empty_result() | {"mapping_reasons": []}
            time.sleep(1.5 * (attempt + 1))

    return _empty_result() | {"mapping_reasons": []}


def enrich_papers_batch(
    papers: list[dict],
    delay: float = 1.0,
    model: str | None = None,
    limit: int | None = None,
) -> list[dict]:
    result: list[dict] = []
    total = len(papers) if limit is None else min(len(papers), limit)
    for index, paper in enumerate(papers):
        if limit is not None and index >= limit:
            result.append(paper)
            continue
        print(f"[AI] {index + 1}/{total} {paper.get('title', '')[:72]}")
        enrichment = enrich_paper(
            title=str(paper.get("title", "")),
            abstract=str(paper.get("abstract", "")),
            category=str(paper.get("category", "")),
            model=model,
        )
        existing_keywords = set(paper.get("keywords") or [])
        for keyword in enrichment["keywords_en"]:
            existing_keywords.add(keyword)
        paper["keywords"] = sorted(existing_keywords)
        if enrichment["keywords_cn"]:
            paper["keywordsCn"] = enrichment["keywords_cn"]
        if enrichment["knowledge_hints"]:
            paper["knowledgeHints"] = enrichment["knowledge_hints"]
        if enrichment["topic_hints"]:
            paper["topicHints"] = enrichment["topic_hints"]
        if enrichment["summary_cn"]:
            paper["summaryCn"] = enrichment["summary_cn"]
        result.append(paper)
        if index < total - 1:
            time.sleep(max(0.0, delay))
    return result


def map_papers_to_knowledge_batch(
    papers: list[dict],
    delay: float = 1.0,
    model: str | None = None,
    limit: int | None = None,
) -> list[dict]:
    result: list[dict] = []
    total = len(papers) if limit is None else min(len(papers), limit)
    for index, paper in enumerate(papers):
        if limit is not None and index >= limit:
            result.append(paper)
            continue
        print(f"[AI] 映射知识点 {index + 1}/{total} {paper.get('title', '')[:72]}")
        candidates = select_knowledge_candidates(
            title=str(paper.get("title", "")),
            abstract=str(paper.get("abstract", "")),
            keywords=paper.get("keywords") or [],
            seed_hints=paper.get("knowledgeHints") or [],
        )
        mapping = map_paper_to_knowledge(paper, candidates=candidates, model=model)
        if mapping["knowledge_hints"]:
            paper["knowledgeHints"] = mapping["knowledge_hints"]
        if mapping["topic_hints"]:
            existing_topics = set(paper.get("topicHints") or [])
            existing_topics.update(mapping["topic_hints"])
            paper["topicHints"] = sorted(existing_topics)
        if mapping["summary_cn"] and not paper.get("summaryCn"):
            paper["summaryCn"] = mapping["summary_cn"]
        if mapping.get("mapping_reasons"):
            paper["mappingReasons"] = mapping["mapping_reasons"]
        result.append(paper)
        if index < total - 1:
            time.sleep(max(0.0, delay))
    return result
