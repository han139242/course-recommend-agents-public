from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Iterable

SCRAPER_ROOT = Path(__file__).resolve().parent
REPO_ROOT = SCRAPER_ROOT.parents[1]
TXT2NEO4J_ROOT = REPO_ROOT / "pipelines" / "txt2neo4j"
CURATED_BATCH_DIR = TXT2NEO4J_ROOT / "config" / "curated_batches"
CURATED_SEED_PATH = TXT2NEO4J_ROOT / "config" / "curated_seed_graph.json"
SYNONYM_MAP_PATH = TXT2NEO4J_ROOT / "dictionaries" / "synonym_map.json"
TERM_DICTIONARY_PATH = TXT2NEO4J_ROOT / "dictionaries" / "term_dictionary.json"


@dataclass(frozen=True)
class KnowledgeCandidate:
    name: str
    course: str = ""
    entity_type: str = "concept"
    description: str = ""
    aliases: tuple[str, ...] = field(default_factory=tuple)


def normalize_space(text: str) -> str:
    return re.sub(r"\s+", " ", str(text or "")).strip()


def normalize_key(text: str) -> str:
    return normalize_space(text).casefold()


def _load_json(path: Path, default):
    if not path.exists():
        return default
    return json.loads(path.read_text(encoding="utf-8"))


def _dedupe(values: Iterable[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for value in values:
        item = normalize_space(value)
        if not item:
            continue
        key = normalize_key(item)
        if key in seen:
            continue
        seen.add(key)
        result.append(item)
    return result


def load_synonym_aliases() -> dict[str, set[str]]:
    aliases: dict[str, set[str]] = {}
    synonym_map = _load_json(SYNONYM_MAP_PATH, {})
    for alias, canonical in synonym_map.items():
        canonical_name = normalize_space(canonical)
        if not canonical_name:
            continue
        aliases.setdefault(canonical_name, set()).add(normalize_space(alias))
        aliases[canonical_name].add(canonical_name)

    for item in _load_json(TERM_DICTIONARY_PATH, []):
        canonical_name = normalize_space(item.get("canonical_name", ""))
        if not canonical_name:
            continue
        bucket = aliases.setdefault(canonical_name, set())
        bucket.add(canonical_name)
        for alias in item.get("aliases", []):
            if normalize_space(alias):
                bucket.add(normalize_space(alias))
    return aliases


def load_knowledge_catalog(limit: int | None = None) -> list[KnowledgeCandidate]:
    alias_map = load_synonym_aliases()
    by_name: dict[str, KnowledgeCandidate] = {}
    seed_paths = [CURATED_SEED_PATH, *sorted(CURATED_BATCH_DIR.glob("*.json"))]

    for seed_path in seed_paths:
        payload = _load_json(seed_path, {})
        course = normalize_space(payload.get("course_title") or payload.get("source") or seed_path.stem)
        for item in payload.get("nodes", []):
            name = normalize_space(item.get("canonical_name") or item.get("name") or "")
            if not name:
                continue
            aliases = _dedupe([name, *item.get("aliases", []), *alias_map.get(name, set())])
            candidate = KnowledgeCandidate(
                name=name,
                course=course,
                entity_type=normalize_space(item.get("entity_type") or "concept"),
                description=normalize_space(item.get("description") or ""),
                aliases=tuple(aliases),
            )
            old = by_name.get(name)
            if old is None or len(candidate.description) > len(old.description):
                by_name[name] = candidate

    candidates = sorted(by_name.values(), key=lambda item: (item.course, item.name))
    return candidates[:limit] if limit else candidates


def score_candidate(candidate: KnowledgeCandidate, text: str, seed_hints: Iterable[str] = ()) -> float:
    lookup = normalize_key(text)
    hint_lookup = " ".join(normalize_key(item) for item in seed_hints)
    score = 0.0
    for alias in candidate.aliases or (candidate.name,):
        alias_key = normalize_key(alias)
        if not alias_key or len(alias_key) < 2:
            continue
        if alias_key in hint_lookup:
            score += 8.0
        if alias_key in lookup:
            score += 4.0 + min(len(alias_key), 24) / 8
    for token in normalize_key(candidate.description).split():
        if len(token) > 3 and token in lookup:
            score += 0.3
    return score


def select_knowledge_candidates(
    title: str,
    abstract: str,
    keywords: Iterable[str] = (),
    seed_hints: Iterable[str] = (),
    max_candidates: int = 50,
    catalog: list[KnowledgeCandidate] | None = None,
) -> list[KnowledgeCandidate]:
    catalog = catalog or load_knowledge_catalog()
    text = " ".join([title, abstract, *keywords])
    scored = [
        (score_candidate(candidate, text, seed_hints), candidate)
        for candidate in catalog
    ]
    selected = [candidate for score, candidate in sorted(scored, key=lambda item: (-item[0], item[1].name)) if score > 0]

    if len(selected) < 12:
        foundation_names = {
            "Transformer",
            "注意力机制",
            "大语言模型",
            "机器学习",
            "深度学习",
            "强化学习",
            "知识图谱",
            "检索增强生成",
            "图神经网络",
            "多模态学习",
            "AI 安全",
            "自然语言处理",
            "计算机视觉",
        }
        existing = {item.name for item in selected}
        selected.extend(
            candidate
            for candidate in catalog
            if candidate.name in foundation_names and candidate.name not in existing
        )

    return selected[:max_candidates]


def candidates_to_prompt(candidates: list[KnowledgeCandidate]) -> str:
    lines = []
    for index, item in enumerate(candidates, start=1):
        aliases = "、".join(item.aliases[:4])
        desc = f"；说明：{item.description[:80]}" if item.description else ""
        lines.append(f"{index}. {item.name}（课程：{item.course or '未分类'}；别名：{aliases}{desc}）")
    return "\n".join(lines)


def candidate_name_set(candidates: list[KnowledgeCandidate]) -> set[str]:
    names: set[str] = set()
    for item in candidates:
        names.add(item.name)
        names.update(item.aliases)
    return {normalize_key(item) for item in names if normalize_space(item)}
