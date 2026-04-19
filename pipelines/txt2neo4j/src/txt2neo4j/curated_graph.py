from __future__ import annotations

import json
from hashlib import sha1
from pathlib import Path

from neo4j import GraphDatabase

from .neo4j_import import SUPPORTED_RELATION_TYPES, create_constraints
from .validate_curated_batch import validate_seed_path

PIPELINE_ROOT = Path(__file__).resolve().parents[2]
CURATED_SEED_PATH = PIPELINE_ROOT / "config" / "curated_seed_graph.json"
CURATED_BATCH_DIR = PIPELINE_ROOT / "config" / "curated_batches"
REFERENCE_TAXONOMY_PATH = PIPELINE_ROOT / "config" / "reference_taxonomy.json"


def _stable_id(*parts: str) -> str:
    raw = "||".join(parts)
    return sha1(raw.encode("utf-8")).hexdigest()[:16]


def load_json_file(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def resolve_seed_path(seed_name_or_path: str | None) -> Path:
    if not seed_name_or_path:
        return CURATED_SEED_PATH
    raw = Path(seed_name_or_path)
    if raw.is_absolute():
        return raw
    if raw.suffix == ".json":
        return CURATED_BATCH_DIR / raw.name
    return CURATED_BATCH_DIR / f"{seed_name_or_path}.json"


def resolve_seed_paths_from_pattern(pattern: str) -> list[Path]:
    raw_pattern = str(pattern or "").strip()
    if not raw_pattern:
        return []

    raw = Path(raw_pattern)
    if raw.is_absolute():
        base_dir = raw.parent
        glob_pattern = raw.name
    else:
        base_dir = CURATED_BATCH_DIR
        if any(token in raw_pattern for token in "*?[]"):
            glob_pattern = raw_pattern
        elif raw_pattern.endswith(".json"):
            glob_pattern = raw_pattern
        else:
            glob_pattern = f"{raw_pattern}.json"

    matches = sorted(path for path in base_dir.glob(glob_pattern) if path.is_file())
    if not matches:
        raise FileNotFoundError(f"no curated seed files matched pattern: {pattern}")
    return matches


def load_reference_topics() -> set[str]:
    payload = load_json_file(REFERENCE_TAXONOMY_PATH)
    topics: set[str] = set()
    for domain in payload.get("domains", []):
        topics.add(domain["name"])
        topics.update(domain.get("children", []))
    return topics


def load_external_registry_nodes(target_path: Path) -> dict[str, dict]:
    registry: dict[str, dict] = {}
    target_resolved = target_path.resolve()
    paths = [CURATED_SEED_PATH, *sorted(CURATED_BATCH_DIR.glob("*.json"))]
    for path in paths:
        if not path.exists() or path.resolve() == target_resolved:
            continue
        payload = load_json_file(path)
        source = str(payload.get("source", path.stem)).strip()
        course_title = str(payload.get("course_title", "")).strip()
        for node in payload.get("nodes", []):
            canonical_name = str(node.get("canonical_name", "")).strip()
            if not canonical_name or canonical_name in registry:
                continue
            registry[canonical_name] = {
                "canonical_name": canonical_name,
                "entity_type": str(node.get("entity_type", "")).strip(),
                "description": str(node.get("description", "")).strip(),
                "source": source,
                "course_title": course_title,
            }
    return registry


def fetch_existing_nodes(driver, database: str) -> dict[str, dict]:
    query = """
    MATCH (n:KnowledgeNode)
    RETURN n.node_id AS node_id, n.name AS name, n.canonical_name AS canonical_name
    """
    rows = driver.execute_query(query, database_=database).records
    mapping: dict[str, dict] = {}
    for row in rows:
        node = {
            "node_id": row["node_id"],
            "name": row["name"] or "",
            "canonical_name": row["canonical_name"] or row["name"] or "",
        }
        for key in {node["name"], node["canonical_name"]}:
            if key:
                mapping[key] = node
    return mapping


def build_coverage_report(existing_names: set[str]) -> dict:
    reference_topics = sorted(load_reference_topics())
    covered = [name for name in reference_topics if name in existing_names]
    missing = [name for name in reference_topics if name not in existing_names]
    return {
        "reference_topic_count": len(reference_topics),
        "covered_topic_count": len(covered),
        "missing_topic_count": len(missing),
        "coverage_ratio": round(len(covered) / len(reference_topics), 4) if reference_topics else 1.0,
        "covered_topics": covered,
        "missing_topics": missing,
    }


def build_curated_rows(existing_lookup: dict[str, dict], seed_path: Path) -> tuple[list[dict], list[dict], dict]:
    seed = load_json_file(seed_path)
    source_tag = seed["source"]
    course_title = seed.get("course_title", "人工智能基础知识骨架")
    external_registry = load_external_registry_nodes(seed_path)
    node_rows: list[dict] = []
    relation_rows: list[dict] = []
    canonical_to_id: dict[str, str] = {}

    for item in seed.get("nodes", []):
        canonical_name = item["canonical_name"]
        existing = existing_lookup.get(canonical_name)
        registry_item = external_registry.get(canonical_name)
        node_id = existing["node_id"] if existing else _stable_id("entity", canonical_name)
        canonical_to_id[canonical_name] = node_id
        resolved_source = registry_item["source"] if registry_item else source_tag
        resolved_course_title = registry_item.get("course_title") if registry_item else course_title
        resolved_entity_type = registry_item.get("entity_type") if registry_item else item["entity_type"]
        resolved_description = registry_item.get("description") if registry_item else item.get("description", "")
        node_rows.append(
            {
                "node_id": node_id,
                "name": canonical_name,
                "raw_name": canonical_name,
                "canonical_name": canonical_name,
                "entity_type": resolved_entity_type,
                "source_structure_id": f"curated::{canonical_name}",
                "source_path": f"curated/{resolved_source}",
                "chapter_title": resolved_course_title,
                "description": resolved_description,
                "aliases": [canonical_name],
                "facets": [],
                "confidence": 1.0,
                "review_status": "approved",
                "quality_issues": [],
                "source_line_no": 0,
                "source_kind": "curated",
                "quality_tier": "gold",
                "curated_source": resolved_source,
            }
        )

    for source_name, target_name, relation_type in seed.get("relations", []):
        if relation_type not in SUPPORTED_RELATION_TYPES:
            continue
        source_id = canonical_to_id.get(source_name) or existing_lookup.get(source_name, {}).get("node_id")
        target_id = canonical_to_id.get(target_name) or existing_lookup.get(target_name, {}).get("node_id")
        if not source_id or not target_id or source_id == target_id:
            continue
        relation_rows.append(
            {
                "relation_id": _stable_id(relation_type, source_id, target_id),
                "source_id": source_id,
                "target_id": target_id,
                "relation_type": relation_type,
                "confidence": 1.0,
                "source_text": "curated_foundation_taxonomy",
                "source_path": f"curated/{source_tag}",
            }
        )

    return node_rows, relation_rows, seed


def merge_curated_graph(
    uri: str,
    username: str,
    password: str,
    database: str = "neo4j",
    seed_name_or_path: str | None = None,
) -> dict:
    driver = GraphDatabase.driver(uri, auth=(username, password))
    try:
        seed_path = resolve_seed_path(seed_name_or_path)
        validation = validate_seed_path(seed_path)
        if not validation["valid"]:
            raise ValueError(
                "Curated seed validation failed:\n"
                + json.dumps(validation, ensure_ascii=False, indent=2)
            )
        create_constraints(driver)
        existing_lookup = fetch_existing_nodes(driver, database)
        existing_canonical_names = {
            node["canonical_name"]
            for node in existing_lookup.values()
            if node.get("canonical_name")
        }
        before_names = {
            node["canonical_name"]
            for node in existing_lookup.values()
            if node.get("canonical_name")
        } | {
            node["name"]
            for node in existing_lookup.values()
            if node.get("name")
        }
        coverage_before = build_coverage_report(before_names)
        node_rows, relation_rows, seed = build_curated_rows(existing_lookup, seed_path)

        with driver.session(database=database) as session:
            if node_rows:
                session.run(
                    """
                    UNWIND $rows AS row
                    MERGE (n:KnowledgeNode {node_id: row.node_id})
                    ON CREATE SET n += row
                    SET
                      n.name = coalesce(n.name, row.name),
                      n.raw_name = coalesce(n.raw_name, row.raw_name),
                      n.canonical_name = coalesce(n.canonical_name, row.canonical_name),
                      n.entity_type = coalesce(n.entity_type, row.entity_type),
                      n.source_structure_id = coalesce(n.source_structure_id, row.source_structure_id),
                      n.source_path = coalesce(n.source_path, row.source_path),
                      n.chapter_title = CASE
                        WHEN trim(coalesce(n.chapter_title, '')) = '' THEN row.chapter_title
                        ELSE n.chapter_title
                      END,
                      n.description = CASE
                        WHEN trim(coalesce(n.description, '')) = '' THEN row.description
                        ELSE n.description
                      END,
                      n.aliases = CASE
                        WHEN size(coalesce(n.aliases, [])) = 0 THEN row.aliases
                        ELSE n.aliases
                      END,
                      n.facets = CASE
                        WHEN size(coalesce(n.facets, [])) = 0 THEN row.facets
                        ELSE n.facets
                      END,
                      n.confidence = coalesce(n.confidence, row.confidence),
                      n.review_status = coalesce(n.review_status, row.review_status),
                      n.quality_issues = CASE
                        WHEN size(coalesce(n.quality_issues, [])) = 0 THEN row.quality_issues
                        ELSE n.quality_issues
                      END,
                      n.source_line_no = coalesce(n.source_line_no, row.source_line_no),
                      n.source_kind = coalesce(n.source_kind, row.source_kind),
                      n.quality_tier = coalesce(n.quality_tier, row.quality_tier),
                      n.curated_source = coalesce(n.curated_source, row.curated_source)
                    """,
                    rows=node_rows,
                )

            for relation_type in SUPPORTED_RELATION_TYPES:
                typed_rows = [row for row in relation_rows if row["relation_type"] == relation_type]
                if not typed_rows:
                    continue
                query = f"""
                UNWIND $rows AS row
                MATCH (a:KnowledgeNode {{node_id: row.source_id}})
                MATCH (b:KnowledgeNode {{node_id: row.target_id}})
                MERGE (a)-[r:{relation_type}]->(b)
                SET r += row
                """
                session.run(query, rows=typed_rows)

        refreshed_lookup = fetch_existing_nodes(driver, database)
        after_names = {
            node["canonical_name"]
            for node in refreshed_lookup.values()
            if node.get("canonical_name")
        } | {
            node["name"]
            for node in refreshed_lookup.values()
            if node.get("name")
        }
        coverage_after = build_coverage_report(after_names)
        nodes_inserted = sum(1 for row in node_rows if row["canonical_name"] not in existing_canonical_names)
        report = {
            "source": seed["source"],
            "nodes_inserted": nodes_inserted,
            "nodes_overlaid": len(node_rows) - nodes_inserted,
            "relations_merged": len(relation_rows),
            "coverage_before": coverage_before,
            "coverage_after": coverage_after,
        }
        report_dir = PIPELINE_ROOT / "artifacts" / seed["source"]
        report_dir.mkdir(parents=True, exist_ok=True)
        report_path = report_dir / "coverage_audit.json"
        report_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
        return report
    finally:
        driver.close()
