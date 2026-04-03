from __future__ import annotations

import csv
import hashlib
import json
import re
import unicodedata
from collections import Counter, defaultdict
from pathlib import Path
from typing import Iterable

from .models import KnowledgeNode, PipelineBundle, RelationRecord, StructureNode, TextLine

PIPELINE_ROOT = Path(__file__).resolve().parents[2]
CONFIG_PATH = PIPELINE_ROOT / "config" / "pipeline_config.json"
TERM_DICTIONARY_PATH = PIPELINE_ROOT / "dictionaries" / "term_dictionary.json"
SYNONYM_MAP_PATH = PIPELINE_ROOT / "dictionaries" / "synonym_map.json"

CHAPTER_RE = re.compile(r"^第\s*(?P<code>[0-9一二三四五六七八九十百]+)\s*章\s*(?P<title>.+?)(?P<page>\d+)?$")
HEADING_RE = re.compile(
    r"^(?P<code>\d+(?:\.\d+){1,2})\s+(?P<title>.+?)(?P<page>\d+)?$"
)
PURE_NUMBER_RE = re.compile(r"^\d+(?:\.\d+)?$")
URL_RE = re.compile(r"^(https?://|www\.)", re.I)

NOISE_PATTERNS = [
    re.compile(r"^ISBN\b", re.I),
    re.compile(r"^http", re.I),
    re.compile(r"^版权所有"),
    re.compile(r"^物料号"),
    re.compile(r"^咨询电话"),
    re.compile(r"^购书热线"),
]

GENERIC_TITLES = {
    "小结",
    "习题",
    "附录",
    "参考文献",
    "实验指导",
    "内容提要",
    "前言",
}

AMBIGUOUS_SHORT_TITLES = {
    "孕育",
    "形成",
    "发展",
    "编码",
    "选择",
    "交叉",
    "变异",
    "已知",
    "定义",
    "过程",
    "步骤",
    "示例",
}

BOOK_TITLE_STOPWORDS = {
    "本书特色",
    "内容提要",
    "数字课程资源使用说明",
    "前言",
}

EXERCISE_PREFIXES = (
    "已知",
    "设",
    "求",
    "证明",
    "设计",
    "定义谓词",
    "把已知",
    "若已知",
    "例",
)

QUESTION_PREFIXES = (
    "什么是",
    "请",
    "试",
    "简述",
    "说明",
    "将下列",
    "把下列",
    "如何",
    "为什么",
    "哪些",
    "写出",
    "构造",
    "给出",
    "指出",
)

ENTITY_KEYWORDS = {
    "algorithm": ["算法", "搜索", "遗传", "粒子群", "蚁群", "神经网络", "BP", "Hopfield"],
    "method": ["方法", "推理", "表示法", "策略", "原理", "理论", "学习法"],
    "application": ["应用", "系统", "游戏", "自然语言处理", "多智能体"],
    "topic": ["绪论", "知识表示", "专家系统", "机器学习", "人工智能", "智能体"],
}

CORE_ENTITY_HINTS = [
    "算法",
    "网络",
    "表示法",
    "系统",
    "学习",
    "推理",
    "搜索",
    "模型",
    "知识",
    "智能",
    "神经",
    "谓词",
    "框架",
    "产生式",
    "语言处理",
]

FACET_PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    ("概念", re.compile(r"^(?P<core>.+?)的概念$")),
    ("定义", re.compile(r"^(?P<core>.+?)的定义$")),
    ("特征", re.compile(r"^(?P<core>.+?)的特征$")),
    ("特性", re.compile(r"^(?P<core>.+?)的特性$")),
    ("特点", re.compile(r"^(?P<core>.+?)的特点$")),
    ("性质", re.compile(r"^(?P<core>.+?)的性质$")),
    ("结构", re.compile(r"^(?P<core>.+?)的一般结构$")),
    ("步骤", re.compile(r"^(?P<core>.+?)的一般步骤$")),
    ("原理", re.compile(r"^(?P<core>.+?)的基本原理$")),
    ("思想", re.compile(r"^(?P<core>.+?)的基本思想$")),
    ("参数分析", re.compile(r"^(?P<core>.+?)的参数分析$")),
    ("应用", re.compile(r"^(?P<core>.+?)的应用(?:领域)?$")),
]

RELATION_PATTERNS = [
    ("USES", re.compile(r"(?P<src>.+?)基于(?P<dst>.+?)")),
    ("APPLIES_TO", re.compile(r"(?P<src>.+?)用于(?P<dst>.+?)")),
    ("PART_OF", re.compile(r"(?P<src>.+?)是(?P<dst>.+?)的一种")),
    ("PREREQUISITE_OF", re.compile(r"(?P<dst>.+?)是(?P<src>.+?)的基础")),
]


def load_json_file(path: Path, default):
    if not path.exists():
        return default
    return json.loads(path.read_text(encoding="utf-8"))


PIPELINE_CONFIG = load_json_file(CONFIG_PATH, {})
TERM_DICTIONARY = load_json_file(TERM_DICTIONARY_PATH, [])
SYNONYM_MAP = {
    normalize_key: normalize_value
    for normalize_key, normalize_value in (
        (unicodedata.normalize("NFKC", key).strip().lower(), value)
        for key, value in load_json_file(SYNONYM_MAP_PATH, {}).items()
    )
}
TERM_ENTITY_TYPES = {
    unicodedata.normalize("NFKC", item["canonical_name"]).strip(): item["entity_type"]
    for item in TERM_DICTIONARY
}
QUALITY_CONFIG = PIPELINE_CONFIG.get("quality", {})
RELATED_CONFIG = PIPELINE_CONFIG.get("related_to", {})
FILTER_CONFIG = PIPELINE_CONFIG.get("filters", {})


def _stable_id(*parts: str) -> str:
    raw = "||".join(parts)
    return hashlib.sha1(raw.encode("utf-8")).hexdigest()[:16]


def normalize_name(text: str) -> str:
    normalized = unicodedata.normalize("NFKC", text)
    normalized = normalized.replace("＊", "*").replace("·", "").replace("：", ":")
    normalized = re.sub(r"\s+", " ", normalized)
    return normalized.strip(" \t\r\n-:：,，。")


def normalize_lookup_key(text: str) -> str:
    return normalize_name(text).lower()


def canonicalize_with_dictionary(name: str) -> str:
    normalized = normalize_name(name)
    lookup_key = normalize_lookup_key(normalized)
    return SYNONYM_MAP.get(lookup_key, normalized)


def extract_book_title(raw_text: str, fallback_title: str) -> str:
    candidate_scores: Counter[str] = Counter()
    lines = raw_text.splitlines()

    for raw_line in lines[:180]:
        line = normalize_name(raw_line)
        if not line:
            continue
        if CHAPTER_RE.match(line):
            break
        if line in BOOK_TITLE_STOPWORDS:
            continue
        if URL_RE.match(line) or PURE_NUMBER_RE.match(line):
            continue
        if any(pattern.match(line) for pattern in NOISE_PATTERNS):
            continue
        if len(line) < 2 or len(line) > 30:
            continue
        if re.search(r"[0-9]{4}-[0-9]{2}-[0-9]{2}", line):
            continue
        if "提取文字" in line:
            continue
        if not re.search(r"[\u4e00-\u9fffA-Za-z]", line):
            continue

        score = 1
        if re.fullmatch(r"[\u4e00-\u9fffA-Za-z0-9（）()·《》\- ]{2,30}", line):
            score += 2
        if "人工智能" in line:
            score += 4
        if any(keyword in line for keyword in ("导论", "原理", "基础", "教程", "概论")):
            score += 2

        candidate_scores[line] += score

    if candidate_scores:
        return candidate_scores.most_common(1)[0][0]

    return fallback_title


def is_suspicious_name(name: str) -> list[str]:
    issues: list[str] = []
    if "提取文字" in name or ".txt" in name.lower():
        issues.append("filename_like_name")
    if re.search(r"\d{4}-\d{2}-\d{2}", name):
        issues.append("timestamp_like_name")
    if len(name) <= 1:
        issues.append("too_short")
    if name in GENERIC_TITLES:
        issues.append("generic_heading")
    if len(name) > 28:
        issues.append("long_sentence_like_name")
    return issues


def should_qualify_with_context(title: str, node: StructureNode) -> bool:
    if node.node_type != "subsection":
        return False
    if title in AMBIGUOUS_SHORT_TITLES:
        return True
    if len(title) <= 3:
        return True
    return False


def is_exercise_heading(title: str) -> bool:
    normalized = normalize_name(title)
    if not normalized:
        return False
    if "习题解答" in normalized:
        return True
    if normalized.startswith(QUESTION_PREFIXES):
        return True
    if "?" in normalized or "？" in normalized:
        return True
    if any(phrase in normalized for phrase in ("求证", "请写出", "请说明", "请简述")):
        return True
    if re.search(r"\bIF\b|\bTHEN\b", normalized, re.I):
        return True
    if re.search(r"[=><\{\}\\_]", normalized):
        return True
    if normalized.startswith(EXERCISE_PREFIXES):
        return True
    if len(normalized) > 24 and any(normalized.startswith(prefix) for prefix in EXERCISE_PREFIXES):
        return True
    if "：" in normalized or ":" in normalized:
        if any(normalized.startswith(prefix) for prefix in EXERCISE_PREFIXES):
            return True
    return False


def is_low_value_title(title: str, node: StructureNode, structure_map: dict[str, StructureNode]) -> bool:
    normalized = normalize_name(title)
    if not normalized:
        return True
    if node.node_type == "chapter":
        return True
    if normalized in {"绪论"} or normalized in set(FILTER_CONFIG.get("drop_titles", [])):
        return True
    if any(normalized.startswith(prefix) for prefix in FILTER_CONFIG.get("drop_prefixes", [])):
        return True
    if normalized in {"孕育", "形成", "发展"}:
        parent = structure_map.get(node.parent_id or "")
        if parent and "发展简史" in parent.title:
            return True
    if len(normalized) <= 2 and normalized not in {"命题", "谓词", "搜索", "编码", "交叉", "变异"}:
        return True
    return False


def extract_facet(title: str) -> tuple[str, str | None]:
    normalized = normalize_name(title)
    for facet, pattern in FACET_PATTERNS:
        match = pattern.match(normalized)
        if match:
            core = normalize_name(match.group("core"))
            if core:
                return core, facet
    return normalized, None


def is_meaningful_core_name(name: str) -> bool:
    normalized = normalize_name(name)
    if len(normalized) < 2:
        return False
    if len(normalized) > 24:
        return False
    if re.search(r"[=><\{\}\\_]", normalized):
        return False
    if all(char.isdigit() or char in ".- " for char in normalized):
        return False
    if any(hint in normalized for hint in CORE_ENTITY_HINTS):
        return True
    if len(normalized) >= 3 and re.search(r"[\u4e00-\u9fff]{3,}", normalized):
        return True
    return False


def get_chapter_title(node: StructureNode, structure_map: dict[str, StructureNode]) -> str:
    current = node
    while current.parent_id:
        parent = structure_map.get(current.parent_id)
        if not parent:
            break
        if parent.node_type == "chapter":
            return parent.title
        current = parent
    return ""


def build_display_name(node: StructureNode, structure_map: dict[str, StructureNode]) -> tuple[str, list[str]]:
    title = normalize_name(node.title)
    issues = is_suspicious_name(title)

    if should_qualify_with_context(title, node) and node.parent_id:
        parent = structure_map.get(node.parent_id)
        if parent:
            parent_title = normalize_name(parent.title)
            qualified = normalize_name(f"{parent_title}·{title}")
            return qualified, issues

    return title, issues


def clean_text_lines(raw_text: str) -> list[TextLine]:
    lines: list[TextLine] = []
    body_started = False

    for line_no, raw_line in enumerate(raw_text.splitlines(), start=1):
        line = normalize_name(raw_line)
        line = line.replace("http: //", "http://").replace("http:/ /", "http://")
        line = re.sub(r"\s{2,}", " ", line).strip()

        if not line:
            continue

        if not body_started:
            if CHAPTER_RE.match(line) and "第1章" in line:
                body_started = True
            else:
                continue

        if URL_RE.match(line) or PURE_NUMBER_RE.match(line):
            continue

        if any(pattern.match(line) for pattern in NOISE_PATTERNS):
            continue

        if len(line) == 1 and line not in {"A", "B"}:
            continue

        lines.append(TextLine(line_no=line_no, text=line))

    return lines


def parse_structure(cleaned_lines: list[TextLine], source_name: str, book_title: str) -> list[StructureNode]:
    book_id = _stable_id("book", book_title)
    book_node = StructureNode(
        node_id=book_id,
        node_type="book",
        level=0,
        code="BOOK",
        title=book_title,
        canonical_name=normalize_name(book_title),
        parent_id=None,
        path=book_title,
        line_no=1,
    )

    nodes: list[StructureNode] = [book_node]
    active_ids: dict[int, str] = {0: book_id}

    for line in cleaned_lines:
        chapter_match = CHAPTER_RE.match(line.text)
        heading_match = HEADING_RE.match(line.text)

        if chapter_match:
            code = f"第{chapter_match.group('code')}章"
            title = normalize_name(chapter_match.group("title"))
            if title in GENERIC_TITLES or "习题解答" in title or "实验指导书" in title:
                continue
            node = StructureNode(
                node_id=_stable_id("chapter", code, title),
                node_type="chapter",
                level=1,
                code=code,
                title=title,
                canonical_name=normalize_name(title),
                parent_id=book_id,
                path=f"{book_title}/{code} {title}",
                line_no=line.line_no,
            )
            nodes.append(node)
            active_ids = {0: book_id, 1: node.node_id}
            continue

        if heading_match:
            code = heading_match.group("code")
            title = normalize_name(heading_match.group("title"))
            if title in GENERIC_TITLES:
                continue
            if is_exercise_heading(title):
                continue

            depth = code.count(".")
            level = 2 if depth == 1 else 3
            node_type = "section" if level == 2 else "subsection"
            parent_id = active_ids.get(level - 1) or active_ids.get(1) or book_id
            parent_path = next((item.path for item in nodes if item.node_id == parent_id), book_title)
            node = StructureNode(
                node_id=_stable_id(node_type, code, title),
                node_type=node_type,
                level=level,
                code=code,
                title=title,
                canonical_name=normalize_name(title),
                parent_id=parent_id,
                path=f"{parent_path}/{code} {title}",
                line_no=line.line_no,
            )
            nodes.append(node)
            active_ids[level] = node.node_id
            if level == 2 and 3 in active_ids:
                del active_ids[3]
            continue

        current_node_id = active_ids.get(3) or active_ids.get(2) or active_ids.get(1)
        if current_node_id:
            for node in reversed(nodes):
                if node.node_id == current_node_id:
                    node.content.append(line.text)
                    break

    deduped: dict[tuple[str, str], StructureNode] = {}
    for node in nodes:
        key = (node.code, node.canonical_name)
        if node.node_type == "book":
            deduped[key] = node
            continue

        if key not in deduped:
            deduped[key] = node
        else:
            deduped[key].content.extend(node.content)

    return list(deduped.values())


def classify_entity_type(title: str) -> str:
    normalized = canonicalize_with_dictionary(title)
    if normalized in TERM_ENTITY_TYPES:
        return TERM_ENTITY_TYPES[normalized]
    for entity_type, keywords in ENTITY_KEYWORDS.items():
        if any(keyword.lower() in normalized.lower() for keyword in keywords):
            return entity_type
    return "concept"


def summarize_content(content: Iterable[str]) -> str:
    text = " ".join(item.strip() for item in content if item.strip())
    text = re.sub(r"\s+", " ", text)
    if len(text) > 180:
        return text[:177] + "..."
    return text


def build_knowledge_nodes(
    structure_nodes: list[StructureNode],
) -> tuple[list[KnowledgeNode], dict[str, str]]:
    results: list[KnowledgeNode] = []
    structure_map = {node.node_id: node for node in structure_nodes}
    structure_to_entity: dict[str, str] = {}

    for node in structure_nodes:
        if node.node_type in {"book", "chapter"}:
            continue
        if node.title in GENERIC_TITLES:
            continue
        if is_exercise_heading(node.title):
            continue
        if is_low_value_title(node.title, node, structure_map):
            continue

        display_name, issues = build_display_name(node, structure_map)
        core_name, facet = extract_facet(display_name)
        canonical_name = canonicalize_with_dictionary(core_name)
        if not is_meaningful_core_name(canonical_name):
            continue

        chapter_title = get_chapter_title(node, structure_map)
        quality_issues = list(issues)
        if facet:
            quality_issues.append("facet_merged")

        review_status = "review" if any(issue in quality_issues for issue in {"filename_like_name", "timestamp_like_name"}) else "approved"
        confidence = 0.95 if node.node_type == "subsection" else 0.9

        entity = KnowledgeNode(
            node_id=_stable_id("entity", canonical_name),
            name=canonical_name,
            raw_name=node.title,
            canonical_name=canonical_name,
            entity_type=classify_entity_type(canonical_name),
            source_structure_id=node.node_id,
            source_path=node.path,
            chapter_title=chapter_title,
            description=summarize_content(node.content),
            aliases=sorted(set([node.title, canonical_name])),
            facets=[facet] if facet else [],
            confidence=confidence,
            review_status=review_status,
            quality_issues=sorted(set(quality_issues)),
            source_line_no=node.line_no,
        )
        results.append(entity)
        structure_to_entity[node.node_id] = entity.node_id

    merged: dict[str, KnowledgeNode] = {}
    for node in results:
        if node.node_id not in merged:
            merged[node.node_id] = node
            continue

        existing = merged[node.node_id]
        existing.aliases = sorted(set(existing.aliases + node.aliases))
        existing.facets = sorted(set(existing.facets + node.facets))
        if len(node.description) > len(existing.description):
            existing.description = node.description
        existing.confidence = max(existing.confidence, node.confidence)
        existing.quality_issues = sorted(set(existing.quality_issues + node.quality_issues))
        if existing.review_status != "review" and node.review_status == "review":
            existing.review_status = "review"

    return list(merged.values()), structure_to_entity


def _find_known_terms(sentence: str, known_terms: list[str]) -> list[str]:
    matched = [term for term in known_terms if len(term) >= 2 and term in sentence]
    return sorted(set(matched), key=lambda item: (-len(item), item))


def infer_parent_relation_type(parent_name: str, child_name: str) -> str:
    if any(keyword in parent_name for keyword in ("算法", "系统", "网络", "模型")):
        return "PART_OF"
    if any(keyword in child_name for keyword in ("应用", "领域")):
        return "APPLIES_TO"
    return "SUBTOPIC_OF"


def should_link_related(parent_name: str, child_name: str) -> bool:
    if parent_name == child_name:
        return False
    if len(parent_name) < 2 or len(child_name) < 2:
        return False
    if parent_name in child_name or child_name in parent_name:
        return True
    if classify_entity_type(parent_name) == classify_entity_type(child_name):
        return True
    overlap = set(parent_name) & set(child_name)
    overlap -= {"的", "与", "及", "和", "法", "型", "式"}
    return len(overlap) >= 2


def build_relations(
    structure_nodes: list[StructureNode],
    knowledge_nodes: list[KnowledgeNode],
    structure_to_entity: dict[str, str],
) -> tuple[list[RelationRecord], list[dict]]:
    relations: list[RelationRecord] = []
    review_edges: list[dict] = []
    entity_by_structure = {node.source_structure_id: node for node in knowledge_nodes}
    entity_by_name = {node.canonical_name: node for node in knowledge_nodes}
    entity_by_id = {node.node_id: node for node in knowledge_nodes}
    known_terms = sorted(entity_by_name.keys(), key=len, reverse=True)
    related_counts: Counter[tuple[str, str]] = Counter()

    for structure in structure_nodes:
        entity = entity_by_structure.get(structure.node_id)
        if entity and structure.parent_id and structure.parent_id in structure_to_entity:
            parent_entity_id = structure_to_entity[structure.parent_id]
            if parent_entity_id != entity.node_id:
                parent_entity = entity_by_id[parent_entity_id]
                relation_type = infer_parent_relation_type(parent_entity.name, entity.name)
                relations.append(
                    RelationRecord(
                        relation_id=_stable_id(relation_type, entity.node_id, parent_entity.node_id),
                        source_id=entity.node_id,
                        target_id=parent_entity.node_id,
                        relation_type=relation_type,
                        confidence=0.96,
                        source_text=structure.title,
                        source_path=structure.path,
                    )
                )

        for content_line in structure.content:
            current_entity_id = structure_to_entity.get(structure.node_id)
            terms = _find_known_terms(content_line, known_terms)
            if len(terms) < 1 or not current_entity_id:
                continue

            current_entity = entity_by_id[current_entity_id]
            for relation_type, pattern in RELATION_PATTERNS:
                match = pattern.search(content_line)
                if not match:
                    continue

                src_term = next((term for term in terms if term in match.groupdict().get("src", "")), None)
                dst_term = next((term for term in terms if term in match.groupdict().get("dst", "")), None)
                if not src_term or not dst_term or src_term == dst_term:
                    continue

                source_node = entity_by_name.get(src_term)
                target_node = entity_by_name.get(dst_term)
                if not source_node or not target_node:
                    continue

                relations.append(
                    RelationRecord(
                        relation_id=_stable_id(relation_type, source_node.node_id, target_node.node_id),
                        source_id=source_node.node_id,
                        target_id=target_node.node_id,
                        relation_type=relation_type,
                        confidence=0.82,
                        source_text=content_line,
                        source_path=structure.path,
                    )
                )

            for term in terms:
                target_node = entity_by_name.get(term)
                if not target_node or target_node.node_id == current_entity.node_id:
                    continue

                pair = tuple(sorted((current_entity.node_id, target_node.node_id)))
                related_counts[pair] += 1

    for (left_id, right_id), count in related_counts.items():
        left = entity_by_id[left_id]
        right = entity_by_id[right_id]
        reasons: list[str] = []
        min_shared_mentions = RELATED_CONFIG.get("min_shared_mentions", 4)
        if count < min_shared_mentions:
            reasons.append("shared_mentions_below_threshold")
        if RELATED_CONFIG.get("require_same_chapter", True):
            if left.chapter_title and right.chapter_title and left.chapter_title != right.chapter_title:
                reasons.append("cross_chapter")
        if RELATED_CONFIG.get("require_overlap_or_same_type", True):
            if not should_link_related(left.name, right.name):
                reasons.append("weak_semantic_overlap")

        confidence = min(0.68 + 0.04 * count, 0.86)
        if reasons:
            review_edges.append(
                {
                    "source_name": left.name,
                    "target_name": right.name,
                    "shared_mentions": count,
                    "source_chapter": left.chapter_title,
                    "target_chapter": right.chapter_title,
                    "confidence": confidence,
                    "reasons": reasons,
                }
            )
            continue

        relations.append(
            RelationRecord(
                relation_id=_stable_id("related", left_id, right_id),
                source_id=left_id,
                target_id=right_id,
                relation_type="RELATED_TO",
                confidence=max(confidence, RELATED_CONFIG.get("min_confidence", 0.84)),
                source_text=f"co_occurrence:{count}",
                source_path=f"{left.chapter_title}|{right.chapter_title}",
            )
        )

    deduped: dict[tuple[str, str, str], RelationRecord] = {}
    for relation in relations:
        key = (relation.source_id, relation.target_id, relation.relation_type)
        if relation.source_id == relation.target_id:
            continue
        if relation.confidence < 0.8:
            continue
        if key not in deduped or relation.confidence > deduped[key].confidence:
            deduped[key] = relation

    return list(deduped.values()), review_edges


def build_quality_report(
    structure_nodes: list[StructureNode],
    knowledge_nodes: list[KnowledgeNode],
    relations: list[RelationRecord],
    review_edges: list[dict],
) -> dict:
    knowledge_counter = Counter(node.canonical_name for node in knowledge_nodes)
    relation_counter = Counter(relation.relation_type for relation in relations)
    relation_index: defaultdict[str, int] = defaultdict(int)

    for relation in relations:
        relation_index[relation.source_id] += 1
        relation_index[relation.target_id] += 1

    low_confidence_nodes = [node.canonical_name for node in knowledge_nodes if node.confidence < 0.8]
    review_required_nodes = [
        {
            "name": node.name,
            "raw_name": node.raw_name,
            "issues": node.quality_issues,
            "path": node.source_path,
        }
        for node in knowledge_nodes
        if node.review_status == "review"
    ]
    isolated_nodes = [
        node.canonical_name
        for node in knowledge_nodes
        if relation_index.get(node.node_id, 0) <= 1
    ]

    return {
        "structure_nodes": len(structure_nodes),
        "knowledge_nodes": len(knowledge_nodes),
        "relations": len(relations),
        "relation_breakdown": dict(relation_counter),
        "duplicate_name_candidates": [name for name, count in knowledge_counter.items() if count > 1],
        "low_confidence_nodes": low_confidence_nodes,
        "review_required_nodes": review_required_nodes[:100],
        "review_related_edges": review_edges[: QUALITY_CONFIG.get("review_export_limit", 300)],
        "review_related_edge_count": len(review_edges),
        "isolated_nodes": isolated_nodes[:50],
        "quality_gate_passed": len(low_confidence_nodes) == 0 and len(review_required_nodes) == 0 and len(relations) > 0,
    }


def export_bundle(bundle: PipelineBundle, output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    neo4j_dir = output_dir / "neo4j"
    review_dir = output_dir / "review"
    neo4j_dir.mkdir(parents=True, exist_ok=True)
    review_dir.mkdir(parents=True, exist_ok=True)

    (output_dir / "cleaned_lines.json").write_text(
        json.dumps([line.model_dump() for line in bundle.cleaned_lines], ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    (output_dir / "structure_nodes.json").write_text(
        json.dumps([node.model_dump() for node in bundle.structure_nodes], ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    (output_dir / "knowledge_nodes.json").write_text(
        json.dumps([node.model_dump() for node in bundle.knowledge_nodes], ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    (output_dir / "relations.json").write_text(
        json.dumps([relation.model_dump() for relation in bundle.relations], ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    (output_dir / "quality_report.json").write_text(
        json.dumps(bundle.quality_report, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    (review_dir / "review_related_edges.json").write_text(
        json.dumps(bundle.quality_report.get("review_related_edges", []), ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    (review_dir / "review_nodes.json").write_text(
        json.dumps(bundle.quality_report.get("review_required_nodes", []), ensure_ascii=False, indent=2),
        encoding="utf-8",
    )

    with (review_dir / "review_related_edges.csv").open("w", encoding="utf-8", newline="") as file:
        writer = csv.DictWriter(
            file,
            fieldnames=[
                "source_name",
                "target_name",
                "shared_mentions",
                "source_chapter",
                "target_chapter",
                "confidence",
                "reasons",
            ],
        )
        writer.writeheader()
        for item in bundle.quality_report.get("review_related_edges", []):
            writer.writerow(
                {
                    **item,
                    "reasons": "|".join(item.get("reasons", [])),
                }
            )

    with (neo4j_dir / "structure_nodes.csv").open("w", encoding="utf-8", newline="") as file:
        writer = csv.DictWriter(
            file,
            fieldnames=["node_id", "node_type", "level", "code", "title", "canonical_name", "parent_id", "path", "line_no"],
        )
        writer.writeheader()
        for node in bundle.structure_nodes:
            writer.writerow(
                {
                    "node_id": node.node_id,
                    "node_type": node.node_type,
                    "level": node.level,
                    "code": node.code,
                    "title": node.title,
                    "canonical_name": node.canonical_name,
                    "parent_id": node.parent_id or "",
                    "path": node.path,
                    "line_no": node.line_no,
                }
            )

    with (neo4j_dir / "knowledge_nodes.csv").open("w", encoding="utf-8", newline="") as file:
        writer = csv.DictWriter(
            file,
            fieldnames=[
                "node_id",
                "name",
                "raw_name",
                "canonical_name",
                "entity_type",
                "source_structure_id",
                "source_path",
                "chapter_title",
                "description",
                "aliases",
                "facets",
                "confidence",
                "review_status",
                "quality_issues",
                "source_line_no",
            ],
        )
        writer.writeheader()
        for node in bundle.knowledge_nodes:
            writer.writerow(
                {
                    "node_id": node.node_id,
                    "name": node.name,
                    "raw_name": node.raw_name,
                    "canonical_name": node.canonical_name,
                    "entity_type": node.entity_type,
                    "source_structure_id": node.source_structure_id,
                    "source_path": node.source_path,
                    "chapter_title": node.chapter_title,
                    "description": node.description,
                    "aliases": "|".join(node.aliases),
                    "facets": "|".join(node.facets),
                    "confidence": node.confidence,
                    "review_status": node.review_status,
                    "quality_issues": "|".join(node.quality_issues),
                    "source_line_no": node.source_line_no,
                }
            )

    with (neo4j_dir / "relations.csv").open("w", encoding="utf-8", newline="") as file:
        writer = csv.DictWriter(
            file,
            fieldnames=["relation_id", "source_id", "target_id", "relation_type", "confidence", "source_text", "source_path"],
        )
        writer.writeheader()
        for relation in bundle.relations:
            writer.writerow(relation.model_dump())


def run_pipeline(input_path: Path, output_dir: Path) -> PipelineBundle:
    raw_text = input_path.read_text(encoding="utf-8")
    book_title = extract_book_title(raw_text, input_path.stem)
    cleaned_lines = clean_text_lines(raw_text)
    structure_nodes = parse_structure(cleaned_lines, input_path.stem, book_title)
    knowledge_nodes, structure_to_entity = build_knowledge_nodes(structure_nodes)
    relations, review_edges = build_relations(structure_nodes, knowledge_nodes, structure_to_entity)
    quality_report = build_quality_report(structure_nodes, knowledge_nodes, relations, review_edges)

    bundle = PipelineBundle(
        source_path=str(input_path),
        cleaned_lines=cleaned_lines,
        structure_nodes=structure_nodes,
        knowledge_nodes=knowledge_nodes,
        relations=relations,
        quality_report=quality_report,
    )
    export_bundle(bundle, output_dir)
    return bundle
