from __future__ import annotations

import csv
import hashlib
import json
import re
import unicodedata
from collections import Counter, defaultdict
from datetime import UTC, datetime
from pathlib import Path
from typing import Iterable

from .models import (
    AuthorNode,
    DigestKnowledgeLink,
    DigestPaperLink,
    FrontierBundle,
    FrontierDigest,
    FrontierKnowledgeNodeRef,
    FrontierMetadataAuthorInput,
    FrontierMetadataInput,
    FrontierMetadataPaperInput,
    FrontierPaper,
    PaperAuthorLink,
    PaperTopicLink,
    PaperVenueLink,
    ResearchTopic,
    TopicKnowledgeLink,
    VenueNode,
)

PIPELINE_ROOT = Path(__file__).resolve().parents[2]
CURATED_BATCH_DIR = PIPELINE_ROOT / "config" / "curated_batches"
CURATED_SEED_PATH = PIPELINE_ROOT / "config" / "curated_seed_graph.json"
TERM_DICTIONARY_PATH = PIPELINE_ROOT / "dictionaries" / "term_dictionary.json"
SYNONYM_MAP_PATH = PIPELINE_ROOT / "dictionaries" / "synonym_map.json"

MIN_MATCH_SCORE = 3.0
MAX_KNOWLEDGE_MATCHES = 4
MAX_TOPIC_CANDIDATES = 4
MAX_TOPIC_KEYWORDS = 6
MAX_DIGEST_TOPICS = 4
MAX_DIGEST_PAPERS = 5


def _stable_id(*parts: str) -> str:
    raw = "||".join(str(part).strip() for part in parts if str(part).strip())
    return hashlib.sha1(raw.encode("utf-8")).hexdigest()[:16]


def _load_json_file(path: Path, default):
    if not path.exists():
        return default
    return json.loads(path.read_text(encoding="utf-8"))


def normalize_name(text: str) -> str:
    normalized = unicodedata.normalize("NFKC", text or "")
    normalized = re.sub(r"\s+", " ", normalized)
    return normalized.strip(" \t\r\n-:;,.，。；、")


def normalize_lookup_key(text: str) -> str:
    return normalize_name(text).lower()


SYNONYM_MAP = {
    normalize_lookup_key(key): normalize_name(value)
    for key, value in _load_json_file(SYNONYM_MAP_PATH, {}).items()
}
TERM_DICTIONARY = _load_json_file(TERM_DICTIONARY_PATH, [])


def canonicalize_with_dictionary(name: str) -> str:
    normalized = normalize_name(name)
    if not normalized:
        return ""
    return SYNONYM_MAP.get(normalize_lookup_key(normalized), normalized)


def clean_text(text: str) -> str:
    cleaned = normalize_name(text)
    cleaned = cleaned.replace(" .", ".").replace(" ,", ",")
    return cleaned


def clean_list(values: Iterable[str]) -> list[str]:
    seen: set[str] = set()
    cleaned: list[str] = []
    for value in values:
        normalized = clean_text(value)
        if not normalized:
            continue
        lookup = normalize_lookup_key(normalized)
        if lookup in seen:
            continue
        seen.add(lookup)
        cleaned.append(normalized)
    return cleaned


def now_iso() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def shorten(text: str, limit: int = 48) -> str:
    cleaned = clean_text(text)
    if len(cleaned) <= limit:
        return cleaned
    return cleaned[: limit - 3] + "..."


def parse_metadata(input_path: Path) -> FrontierMetadataInput:
    if input_path.suffix.lower() == ".jsonl":
        papers: list[dict] = []
        with input_path.open("r", encoding="utf-8") as handle:
            for line in handle:
                line = line.strip()
                if not line:
                    continue
                papers.append(json.loads(line))
        payload = {"source": input_path.stem, "papers": papers}
    else:
        raw_payload = _load_json_file(input_path, {})
        if isinstance(raw_payload, list):
            payload = {"source": input_path.stem, "papers": raw_payload}
        else:
            payload = raw_payload
    return FrontierMetadataInput.model_validate(payload)


def load_term_aliases() -> dict[str, set[str]]:
    aliases_by_canonical: dict[str, set[str]] = defaultdict(set)
    for item in TERM_DICTIONARY:
        canonical = canonicalize_with_dictionary(item.get("canonical_name", ""))
        if not canonical:
            continue
        aliases_by_canonical[canonical].add(canonical)
        aliases_by_canonical[canonical].add(normalize_name(item.get("canonical_name", "")))
        for alias in item.get("aliases", []):
            normalized = normalize_name(alias)
            if normalized:
                aliases_by_canonical[canonical].add(normalized)
    return aliases_by_canonical


def fallback_knowledge_ref(name: str) -> FrontierKnowledgeNodeRef:
    canonical = canonicalize_with_dictionary(name)
    node_id = _stable_id("entity", canonical)
    return FrontierKnowledgeNodeRef(
        node_id=node_id,
        knowledge_id=node_id,
        name=canonical,
        canonical_name=canonical,
        entity_type="concept",
        source_path="frontier/generated",
        chapter_title="Frontier Topics",
        description="",
        aliases=[canonical],
    )


def load_knowledge_catalog() -> tuple[dict[str, FrontierKnowledgeNodeRef], list[tuple[str, str, str]]]:
    term_aliases = load_term_aliases()
    catalog: dict[str, FrontierKnowledgeNodeRef] = {}
    alias_pairs: set[tuple[str, str, str]] = set()
    seed_paths = [CURATED_SEED_PATH, *sorted(CURATED_BATCH_DIR.glob("*.json"))]

    for seed_path in seed_paths:
        payload = _load_json_file(seed_path, {})
        default_course = clean_text(payload.get("course_title") or payload.get("source") or seed_path.stem)
        for item in payload.get("nodes", []):
            canonical = canonicalize_with_dictionary(item.get("canonical_name", ""))
            if not canonical:
                continue
            node_id = _stable_id("entity", canonical)
            ref = catalog.get(canonical)
            if ref is None:
                ref = FrontierKnowledgeNodeRef(
                    node_id=node_id,
                    knowledge_id=node_id,
                    name=canonical,
                    canonical_name=canonical,
                    entity_type=item.get("entity_type", "concept"),
                    source_path=f"curated/{seed_path.name}",
                    chapter_title=default_course,
                    description=clean_text(item.get("description", "")),
                    aliases=[canonical],
                )
                catalog[canonical] = ref
            else:
                if not ref.chapter_title and default_course:
                    ref.chapter_title = default_course
                candidate_description = clean_text(item.get("description", ""))
                if len(candidate_description) > len(ref.description):
                    ref.description = candidate_description

            aliases = {
                canonical,
                normalize_name(item.get("canonical_name", "")),
                *term_aliases.get(canonical, set()),
            }
            for alias in clean_list(aliases):
                alias_pairs.add((normalize_lookup_key(alias), alias, canonical))

    alias_entries = sorted(alias_pairs, key=lambda item: (-len(item[0]), item[2], item[1]))
    return catalog, alias_entries


def normalize_author(item: str | FrontierMetadataAuthorInput) -> FrontierMetadataAuthorInput:
    if isinstance(item, FrontierMetadataAuthorInput):
        return item
    return FrontierMetadataAuthorInput(name=clean_text(str(item)))


def is_ascii_phrase(text: str) -> bool:
    return bool(text) and all(ord(char) < 128 for char in text)


def match_knowledge_nodes(
    paper: FrontierMetadataPaperInput,
    alias_entries: list[tuple[str, str, str]],
) -> list[str]:
    scores: Counter[str] = Counter()

    def add_text_scores(texts: Iterable[str], weight: float, exact_only: bool = False) -> None:
        for text in texts:
            cleaned = clean_text(text)
            if not cleaned:
                continue
            lookup = normalize_lookup_key(cleaned)
            canonical_hint = SYNONYM_MAP.get(lookup)
            if canonical_hint:
                scores[canonical_hint] += weight + 1.5
            for alias_lookup, alias_display, canonical in alias_entries:
                if len(alias_lookup) < 2:
                    continue
                matched = lookup == alias_lookup if exact_only else alias_lookup in lookup
                if not matched:
                    continue
                bonus = 1.0 if lookup == alias_lookup else min(len(alias_display), 24) / 24
                scores[canonical] += weight + bonus

    add_text_scores(paper.knowledge_hints, weight=5.0, exact_only=False)
    add_text_scores(paper.topic_hints, weight=2.5, exact_only=False)
    add_text_scores(paper.keywords, weight=2.0, exact_only=False)
    add_text_scores([paper.title], weight=2.5, exact_only=False)
    add_text_scores([paper.abstract], weight=1.2, exact_only=False)

    matches: list[str] = []
    selected_scores: dict[str, float] = {}
    for canonical, score in sorted(scores.items(), key=lambda item: (-item[1], -len(item[0]), item[0])):
        if score < MIN_MATCH_SCORE:
            continue
        candidate_key = normalize_lookup_key(canonical)
        should_skip = False
        replacement_index = -1
        for index, selected in enumerate(matches):
            selected_key = normalize_lookup_key(selected)
            selected_score = selected_scores[selected]
            if candidate_key in selected_key and len(selected) >= len(canonical) + 2 and selected_score >= score - 1.0:
                should_skip = True
                break
            if selected_key in candidate_key and len(canonical) >= len(selected) + 2 and score >= selected_score - 1.0:
                replacement_index = index
                break
        if should_skip:
            continue
        if replacement_index >= 0:
            replaced = matches[replacement_index]
            matches[replacement_index] = canonical
            selected_scores.pop(replaced, None)
            selected_scores[canonical] = score
            continue
        matches.append(canonical)
        selected_scores[canonical] = score
    if matches:
        return matches[:MAX_KNOWLEDGE_MATCHES]

    fallback_matches = [
        canonicalize_with_dictionary(hint)
        for hint in paper.knowledge_hints
        if canonicalize_with_dictionary(hint)
    ]
    return clean_list(fallback_matches)[:MAX_KNOWLEDGE_MATCHES]


def derive_topic_names(paper: FrontierMetadataPaperInput, linked_knowledge_names: list[str]) -> list[str]:
    topic_candidates: list[str] = []
    topic_candidates.extend(clean_list(paper.topic_hints))
    topic_candidates.extend(clean_list(paper.keywords))
    if not topic_candidates:
        topic_candidates.extend(linked_knowledge_names)

    normalized_topics: list[str] = []
    for candidate in topic_candidates:
        normalized = clean_text(candidate)
        if not normalized or len(normalized) < 2:
            continue
        canonical = canonicalize_with_dictionary(normalized)
        if canonical in linked_knowledge_names:
            normalized_topics.append(canonical)
            continue
        if is_ascii_phrase(normalized):
            normalized_topics.append(normalized)
        else:
            normalized_topics.append(canonical or normalized)
    topics = clean_list(normalized_topics)
    if not topics:
        return linked_knowledge_names[:MAX_TOPIC_CANDIDATES]
    return topics[:MAX_TOPIC_CANDIDATES]


def build_topic_summary(
    topic_name: str,
    linked_knowledge_names: list[str],
    keywords: list[str],
    paper_titles: list[str],
) -> str:
    parts = [f"{topic_name} 是围绕 { '、'.join(linked_knowledge_names[:2]) or '课程知识点' } 延展出的前沿主题。"]
    if keywords:
        parts.append(f"高频关键词包括 { '、'.join(keywords[:3]) }。")
    if paper_titles:
        titles = "、".join(f"《{shorten(title, 30)}》" for title in paper_titles[:2])
        parts.append(f"代表论文包括 {titles}。")
    return "".join(parts)


def build_digest_summary(
    knowledge_name: str,
    topic_names: list[str],
    papers: list[FrontierPaper],
) -> str:
    parts = [f"围绕 {knowledge_name}，当前值得关注的前沿方向主要集中在 { '、'.join(topic_names[:MAX_DIGEST_TOPICS]) or knowledge_name }。"]
    if papers:
        titles = "、".join(f"《{shorten(paper.title, 34)}》" for paper in papers[:3])
        parts.append(f"可优先阅读 {titles}。")
        venues = clean_list(paper.venue for paper in papers if paper.venue)
        years = [paper.year for paper in papers if paper.year is not None]
        if venues or years:
            venue_text = f"相关工作常见于 { '、'.join(venues[:2]) }" if venues else ""
            if years:
                year_text = f"{max(years)} 年附近的最新工作"
            else:
                year_text = ""
            if venue_text or year_text:
                parts.append(f"{venue_text}{' 的' if venue_text and year_text else ''}{year_text}。")
    return "".join(parts)


def export_frontier_bundle(bundle: FrontierBundle, output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    neo4j_dir = output_dir / "neo4j"
    neo4j_dir.mkdir(parents=True, exist_ok=True)

    def write_json(name: str, payload) -> None:
        (output_dir / name).write_text(
            json.dumps(payload, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )

    write_json("knowledge_nodes.json", [item.model_dump() for item in bundle.knowledge_nodes])
    write_json("papers.json", [item.model_dump() for item in bundle.papers])
    write_json("topics.json", [item.model_dump() for item in bundle.topics])
    write_json("venues.json", [item.model_dump() for item in bundle.venues])
    write_json("authors.json", [item.model_dump() for item in bundle.authors])
    write_json("digests.json", [item.model_dump() for item in bundle.digests])
    write_json(
        "relationships.json",
        {
            "paper_topic_links": [item.model_dump() for item in bundle.paper_topic_links],
            "paper_venue_links": [item.model_dump() for item in bundle.paper_venue_links],
            "paper_author_links": [item.model_dump() for item in bundle.paper_author_links],
            "topic_knowledge_links": [item.model_dump() for item in bundle.topic_knowledge_links],
            "digest_knowledge_links": [item.model_dump() for item in bundle.digest_knowledge_links],
            "digest_paper_links": [item.model_dump() for item in bundle.digest_paper_links],
        },
    )
    write_json("report.json", bundle.report)

    def write_csv(name: str, rows: list[dict], fieldnames: list[str]) -> None:
        with (neo4j_dir / name).open("w", encoding="utf-8", newline="") as handle:
            writer = csv.DictWriter(handle, fieldnames=fieldnames)
            writer.writeheader()
            for row in rows:
                writer.writerow(row)

    write_csv(
        "knowledge_nodes.csv",
        [
            {
                "node_id": item.node_id,
                "knowledge_id": item.knowledge_id,
                "name": item.name,
                "canonical_name": item.canonical_name,
                "entity_type": item.entity_type,
                "source_path": item.source_path,
                "chapter_title": item.chapter_title,
                "description": item.description,
                "aliases": "|".join(item.aliases),
            }
            for item in bundle.knowledge_nodes
        ],
        ["node_id", "knowledge_id", "name", "canonical_name", "entity_type", "source_path", "chapter_title", "description", "aliases"],
    )
    write_csv(
        "papers.csv",
        [
            {
                "paper_id": item.paper_id,
                "title": item.title,
                "abstract": item.abstract,
                "year": item.year if item.year is not None else "",
                "venue": item.venue,
                "url": item.url,
                "source": item.source,
                "updated_at": item.updated_at,
                "keywords": "|".join(item.keywords),
            }
            for item in bundle.papers
        ],
        ["paper_id", "title", "abstract", "year", "venue", "url", "source", "updated_at", "keywords"],
    )
    write_csv(
        "topics.csv",
        [
            {
                "topic_id": item.topic_id,
                "name": item.name,
                "canonical_name": item.canonical_name,
                "keywords": "|".join(item.keywords),
                "summary": item.summary,
            }
            for item in bundle.topics
        ],
        ["topic_id", "name", "canonical_name", "keywords", "summary"],
    )
    write_csv(
        "venues.csv",
        [item.model_dump() for item in bundle.venues],
        ["venue_id", "name", "type", "rank"],
    )
    write_csv(
        "authors.csv",
        [item.model_dump() for item in bundle.authors],
        ["author_id", "name", "affiliation"],
    )
    write_csv(
        "digests.csv",
        [item.model_dump() for item in bundle.digests],
        ["digest_id", "knowledge_id", "knowledge_name", "summary", "status", "generated_at"],
    )
    write_csv(
        "paper_topic_links.csv",
        [item.model_dump() for item in bundle.paper_topic_links],
        ["paper_id", "topic_id", "confidence"],
    )
    write_csv(
        "paper_venue_links.csv",
        [item.model_dump() for item in bundle.paper_venue_links],
        ["paper_id", "venue_id"],
    )
    write_csv(
        "paper_author_links.csv",
        [item.model_dump() for item in bundle.paper_author_links],
        ["paper_id", "author_id"],
    )
    write_csv(
        "topic_knowledge_links.csv",
        [item.model_dump() for item in bundle.topic_knowledge_links],
        ["topic_id", "knowledge_node_id"],
    )
    write_csv(
        "digest_knowledge_links.csv",
        [item.model_dump() for item in bundle.digest_knowledge_links],
        ["digest_id", "knowledge_node_id"],
    )
    write_csv(
        "digest_paper_links.csv",
        [item.model_dump() for item in bundle.digest_paper_links],
        ["digest_id", "paper_id"],
    )


def run_frontier_pipeline(input_path: Path, output_dir: Path) -> FrontierBundle:
    metadata = parse_metadata(input_path)
    generated_at = metadata.generated_at or now_iso()
    catalog, alias_entries = load_knowledge_catalog()

    paper_nodes: list[FrontierPaper] = []
    topic_nodes: list[ResearchTopic] = []
    venue_nodes: dict[str, VenueNode] = {}
    author_nodes: dict[str, AuthorNode] = {}
    knowledge_nodes: dict[str, FrontierKnowledgeNodeRef] = {}

    paper_topic_links: set[tuple[str, str]] = set()
    paper_venue_links: set[tuple[str, str]] = set()
    paper_author_links: set[tuple[str, str]] = set()
    topic_knowledge_links: set[tuple[str, str]] = set()
    digest_knowledge_links: set[tuple[str, str]] = set()
    digest_paper_links: set[tuple[str, str]] = set()

    topic_accumulator: dict[str, dict] = {}
    digest_accumulator: dict[str, dict] = defaultdict(
        lambda: {"paper_ids": [], "topic_counts": Counter()}
    )
    unmatched_titles: list[str] = []

    paper_by_id: dict[str, FrontierPaper] = {}

    for item in metadata.papers:
        linked_knowledge_names = match_knowledge_nodes(item, alias_entries)
        if not linked_knowledge_names:
            unmatched_titles.append(item.title)
            continue

        paper_id = _stable_id("paper", metadata.source, item.title, item.url or str(item.year or ""), item.venue)
        paper = FrontierPaper(
            paper_id=paper_id,
            title=clean_text(item.title),
            abstract=clean_text(item.abstract),
            year=item.year,
            venue=clean_text(item.venue),
            url=clean_text(item.url),
            source=clean_text(item.source or metadata.source),
            updated_at=clean_text(item.updated_at or generated_at),
            keywords=clean_list(item.keywords),
        )
        paper_nodes.append(paper)
        paper_by_id[paper.paper_id] = paper

        for knowledge_name in linked_knowledge_names:
            knowledge_nodes[knowledge_name] = catalog.get(knowledge_name) or fallback_knowledge_ref(knowledge_name)

        topic_names = derive_topic_names(item, linked_knowledge_names)
        for topic_name in topic_names:
            topic_key = normalize_lookup_key(topic_name)
            topic_entry = topic_accumulator.setdefault(
                topic_key,
                {
                    "name": topic_name,
                    "keywords": Counter(),
                    "paper_ids": [],
                    "paper_titles": [],
                    "knowledge_names": set(),
                },
            )
            topic_entry["paper_ids"].append(paper.paper_id)
            topic_entry["paper_titles"].append(paper.title)
            topic_entry["knowledge_names"].update(linked_knowledge_names)
            topic_entry["keywords"].update(clean_list([topic_name, *paper.keywords]))

        for knowledge_name in linked_knowledge_names:
            digest_accumulator[knowledge_name]["paper_ids"].append(paper.paper_id)
            digest_accumulator[knowledge_name]["topic_counts"].update(topic_names)

        if paper.venue:
            venue_id = _stable_id("venue", paper.venue)
            venue_nodes[venue_id] = VenueNode(
                venue_id=venue_id,
                name=paper.venue,
                type=clean_text(item.venue_type),
                rank=clean_text(item.venue_rank),
            )
            paper_venue_links.add((paper.paper_id, venue_id))

        for author_input in item.authors:
            author = normalize_author(author_input)
            if not author.name:
                continue
            author_id = _stable_id("author", author.name, author.affiliation)
            author_nodes[author_id] = AuthorNode(
                author_id=author_id,
                name=author.name,
                affiliation=clean_text(author.affiliation),
            )
            paper_author_links.add((paper.paper_id, author_id))

    for topic_key, topic_entry in topic_accumulator.items():
        topic_name = topic_entry["name"]
        topic_id = _stable_id("frontier_topic", topic_name)
        keywords = [
            keyword
            for keyword, _ in topic_entry["keywords"].most_common(MAX_TOPIC_KEYWORDS)
            if normalize_lookup_key(keyword) != topic_key
        ]
        knowledge_names = clean_list(topic_entry["knowledge_names"])
        topic_nodes.append(
            ResearchTopic(
                topic_id=topic_id,
                name=topic_name,
                canonical_name=topic_name,
                keywords=keywords[:MAX_TOPIC_KEYWORDS],
                summary=build_topic_summary(topic_name, knowledge_names, keywords, topic_entry["paper_titles"]),
            )
        )
        for paper_id in clean_list(topic_entry["paper_ids"]):
            paper_topic_links.add((paper_id, topic_id))
        for knowledge_name in knowledge_names:
            knowledge_ref = knowledge_nodes[knowledge_name]
            topic_knowledge_links.add((topic_id, knowledge_ref.node_id))

    digests: list[FrontierDigest] = []
    for knowledge_name, digest_entry in digest_accumulator.items():
        knowledge_ref = knowledge_nodes[knowledge_name]
        ordered_papers = [
            paper_by_id[paper_id]
            for paper_id in clean_list(digest_entry["paper_ids"])
            if paper_id in paper_by_id
        ]
        ordered_papers.sort(key=lambda paper: (paper.year or 0, paper.updated_at, paper.title), reverse=True)
        topic_names = [
            topic_name
            for topic_name, _ in digest_entry["topic_counts"].most_common(MAX_DIGEST_TOPICS)
        ]
        digest_id = _stable_id("frontier_digest", knowledge_ref.node_id, generated_at)
        digests.append(
            FrontierDigest(
                digest_id=digest_id,
                knowledge_id=knowledge_ref.node_id,
                knowledge_name=knowledge_ref.name,
                summary=build_digest_summary(knowledge_ref.name, topic_names, ordered_papers),
                status="ready",
                generated_at=generated_at,
            )
        )
        digest_knowledge_links.add((digest_id, knowledge_ref.node_id))
        for paper in ordered_papers[:MAX_DIGEST_PAPERS]:
            digest_paper_links.add((digest_id, paper.paper_id))

    bundle = FrontierBundle(
        source_path=str(input_path),
        metadata_source=metadata.source,
        generated_at=generated_at,
        knowledge_nodes=sorted(knowledge_nodes.values(), key=lambda item: item.canonical_name),
        papers=sorted(paper_nodes, key=lambda item: (item.year or 0, item.title), reverse=True),
        topics=sorted(topic_nodes, key=lambda item: item.name),
        venues=sorted(venue_nodes.values(), key=lambda item: item.name),
        authors=sorted(author_nodes.values(), key=lambda item: item.name),
        digests=sorted(digests, key=lambda item: item.knowledge_name),
        paper_topic_links=[
            PaperTopicLink(paper_id=paper_id, topic_id=topic_id, confidence=1.0)
            for paper_id, topic_id in sorted(paper_topic_links)
        ],
        paper_venue_links=[
            PaperVenueLink(paper_id=paper_id, venue_id=venue_id)
            for paper_id, venue_id in sorted(paper_venue_links)
        ],
        paper_author_links=[
            PaperAuthorLink(paper_id=paper_id, author_id=author_id)
            for paper_id, author_id in sorted(paper_author_links)
        ],
        topic_knowledge_links=[
            TopicKnowledgeLink(topic_id=topic_id, knowledge_node_id=knowledge_node_id)
            for topic_id, knowledge_node_id in sorted(topic_knowledge_links)
        ],
        digest_knowledge_links=[
            DigestKnowledgeLink(digest_id=digest_id, knowledge_node_id=knowledge_node_id)
            for digest_id, knowledge_node_id in sorted(digest_knowledge_links)
        ],
        digest_paper_links=[
            DigestPaperLink(digest_id=digest_id, paper_id=paper_id)
            for digest_id, paper_id in sorted(digest_paper_links)
        ],
        report={
            "source": metadata.source,
            "generated_at": generated_at,
            "input_papers": len(metadata.papers),
            "linked_papers": len(paper_nodes),
            "unmatched_papers": len(unmatched_titles),
            "unmatched_titles": sorted(unmatched_titles),
            "knowledge_nodes": len(knowledge_nodes),
            "topics": len(topic_nodes),
            "digests": len(digests),
            "venues": len(venue_nodes),
            "authors": len(author_nodes),
            "linked_knowledge_names": sorted(knowledge_nodes.keys()),
        },
    )
    export_frontier_bundle(bundle, output_dir)
    return bundle
