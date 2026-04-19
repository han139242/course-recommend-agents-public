from __future__ import annotations

import argparse
import json
import os
import re
from collections import defaultdict
from dataclasses import dataclass
from datetime import datetime, timezone
from hashlib import sha1
from pathlib import Path

from dotenv import load_dotenv
from neo4j import GraphDatabase

from .curated_graph import CURATED_BATCH_DIR, CURATED_SEED_PATH, load_json_file
from .neo4j_import import SUPPORTED_RELATION_TYPES, create_constraints

PIPELINE_ROOT = Path(__file__).resolve().parents[2]
REFERENCE_TAXONOMY_PATH = PIPELINE_ROOT / "config" / "reference_taxonomy.json"
REPORT_DIR = PIPELINE_ROOT / "artifacts" / "neo4j_normalization"

CURATED_RELATION_SOURCE = "curated/industrial_registry"
NORMALIZATION_SOURCE = "normalized/industrial_v1"

SUSPICIOUS_PREFIXES = ("描述", "举例", "判断", "用", "说明", "证明", "验证", "设有", "试")
SUSPICIOUS_TOKENS = ("?", "？", ",", "，")
MANUAL_PARENT_MAP = {
    "自然语言理解的概念与发展历史": "自然语言处理",
    "语言处理过程的层次": "自然语言处理",
    "鲁宾孙归结原理": "归结原理",
    "归结反演": "归结原理",
    "扫雷机智能游戏开发": "游戏人工智能",
    "游戏中的角色与分类": "游戏人工智能",
    "群智能算法产生的背景": "蚁群算法",
}
DESCRIPTION_BACKFILL_MAP = {
    "进化算法的产生与发展": "介绍进化算法形成背景、核心思想与发展脉络的主题。",
    "专家系统的工作原理": "解释专家系统中知识库、推理机和解释机制如何协同工作的主题。",
    "知识与知识表示": "介绍知识概念以及知识表示方法在人工智能系统中的作用与意义的主题。",
}
SOURCE_PARENT_RULES = (
    ("自然语言处理", "自然语言处理"),
    ("智能体与多智能体系统", "智能体"),
    ("多智能体系统", "多智能体系统"),
    ("知识表示", "知识表示"),
    ("专家系统", "专家系统"),
    ("人工神经网络", "神经网络"),
    ("Hopfield", "神经网络"),
    ("蚁群", "蚁群算法"),
    ("遗传", "遗传算法"),
    ("归结", "归结原理"),
    ("游戏人工智能", "游戏人工智能"),
    ("游戏设计中的应用", "游戏人工智能"),
)
MANUAL_CANONICAL_REDIRECTS = {
    "解释机构": "解释机制",
}
GRAPH_INDEX_STATEMENTS = [
    "CREATE CONSTRAINT knowledge_canonical_unique IF NOT EXISTS FOR (n:KnowledgeNode) REQUIRE n.canonical_name IS UNIQUE",
    "CREATE RANGE INDEX knowledge_entity_type_idx IF NOT EXISTS FOR (n:KnowledgeNode) ON (n.entity_type)",
    "CREATE RANGE INDEX knowledge_source_kind_idx IF NOT EXISTS FOR (n:KnowledgeNode) ON (n.source_kind)",
    "CREATE RANGE INDEX knowledge_taxonomy_domain_idx IF NOT EXISTS FOR (n:KnowledgeNode) ON (n.taxonomy_domain)",
]


@dataclass
class KnowledgeNodeRecord:
    node_id: str
    name: str
    canonical_name: str
    description: str
    entity_type: str
    source_path: str
    source_kind: str
    degree: int
    aliases: list[str]
    legacy_source_paths: list[str]


def _stable_id(*parts: str) -> str:
    raw = "||".join(parts)
    return sha1(raw.encode("utf-8")).hexdigest()[:16]


def normalize_name(value: str) -> str:
    text = str(value or "").strip().lower()
    text = (
        text.replace("（", "(")
        .replace("）", ")")
        .replace("，", ",")
        .replace("：", ":")
    )
    text = re.sub(r"[\s\-_/]+", "", text)
    return text


def unique_list(items: list[str]) -> list[str]:
    result: list[str] = []
    seen: set[str] = set()
    for item in items:
        text = str(item or "").strip()
        if not text or text in seen:
            continue
        seen.add(text)
        result.append(text)
    return result


def choose_primary(nodes: list[KnowledgeNodeRecord]) -> KnowledgeNodeRecord:
    def score(node: KnowledgeNodeRecord) -> tuple[int, int, int, int, int]:
        return (
            1 if node.source_path.startswith("curated/") or node.source_kind == "curated" else 0,
            1 if node.description.strip() else 0,
            node.degree,
            1 if " " not in node.canonical_name else 0,
            -len(node.canonical_name),
        )

    return sorted(nodes, key=score, reverse=True)[0]


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Normalize Neo4j graph to industrial-grade course KG standards.")
    parser.add_argument("--neo4j-uri", default=os.getenv("NEO4J_URI", "bolt://127.0.0.1:7687"))
    parser.add_argument("--neo4j-user", default=os.getenv("NEO4J_USER", os.getenv("NEO4J_USERNAME", "neo4j")))
    parser.add_argument("--neo4j-password", default=os.getenv("NEO4J_PASSWORD", ""))
    parser.add_argument("--neo4j-database", default=os.getenv("NEO4J_DATABASE", "neo4j"))
    return parser


def load_curated_registry() -> tuple[dict[str, dict], set[tuple[str, str, str]]]:
    registry: dict[str, dict] = {}
    relations: set[tuple[str, str, str]] = set()
    paths = [CURATED_SEED_PATH, *sorted(CURATED_BATCH_DIR.glob("*.json"))]
    for path in paths:
        payload = load_json_file(path)
        source = payload["source"]
        course_title = str(payload.get("course_title", "")).strip()
        for node in payload.get("nodes", []):
            canonical_name = str(node.get("canonical_name", "")).strip()
            if not canonical_name:
                continue
            existing = registry.get(canonical_name)
            if not existing:
                registry[canonical_name] = {
                    **node,
                    "canonical_name": canonical_name,
                    "source": source,
                    "course_title": course_title,
                }
                continue
            existing_aliases = existing.get("aliases", [])
            node_aliases = node.get("aliases", [])
            existing["aliases"] = unique_list(existing_aliases + node_aliases + [canonical_name])
            if not str(existing.get("description", "")).strip() and str(node.get("description", "")).strip():
                existing["description"] = node["description"]
            if not str(existing.get("course_title", "")).strip() and course_title:
                existing["course_title"] = course_title
        for relation in payload.get("relations", []):
            if len(relation) != 3:
                continue
            source_name, target_name, relation_type = relation
            if relation_type in SUPPORTED_RELATION_TYPES:
                relations.add((source_name, target_name, relation_type))
    return registry, relations


def load_reference_taxonomy() -> tuple[dict[str, str], list[str], set[str]]:
    payload = load_json_file(REFERENCE_TAXONOMY_PATH)
    topic_to_domain: dict[str, str] = {}
    terms: list[str] = []
    domains: set[str] = set()
    for domain in payload.get("domains", []):
        domain_name = str(domain["name"]).strip()
        domains.add(domain_name)
        topic_to_domain[domain_name] = domain_name
        terms.append(domain_name)
        for child in domain.get("children", []):
            child_name = str(child).strip()
            topic_to_domain[child_name] = domain_name
            terms.append(child_name)
    terms = sorted(set(terms), key=len, reverse=True)
    return topic_to_domain, terms, domains


def fetch_nodes(driver, database: str) -> list[KnowledgeNodeRecord]:
    query = """
    MATCH (n:KnowledgeNode)
    RETURN n.node_id AS node_id,
           coalesce(n.name, '') AS name,
           coalesce(n.canonical_name, n.name, '') AS canonical_name,
           coalesce(n.description, '') AS description,
           coalesce(n.entity_type, 'concept') AS entity_type,
           coalesce(n.source_path, '') AS source_path,
           coalesce(n['source_kind'], '') AS source_kind,
           coalesce(n.aliases, []) AS aliases,
           coalesce(n['legacy_source_paths'], []) AS legacy_source_paths,
           COUNT { (n)--() } AS degree
    """
    rows = driver.execute_query(query, database_=database).records
    result: list[KnowledgeNodeRecord] = []
    for row in rows:
        result.append(
            KnowledgeNodeRecord(
                node_id=row["node_id"],
                name=row["name"],
                canonical_name=row["canonical_name"],
                description=row["description"],
                entity_type=row["entity_type"],
                source_path=row["source_path"],
                source_kind=row["source_kind"],
                degree=int(row["degree"]),
                aliases=[str(item).strip() for item in row["aliases"] if str(item).strip()],
                legacy_source_paths=[str(item).strip() for item in row["legacy_source_paths"] if str(item).strip()],
            )
        )
    return result


def fetch_relation_types(driver, database: str) -> list[str]:
    query = """
    MATCH (a:KnowledgeNode)-[r]-(b:KnowledgeNode)
    RETURN DISTINCT type(r) AS rel_type
    ORDER BY rel_type
    """
    rows = driver.execute_query(query, database_=database).records
    return [str(row["rel_type"]) for row in rows if str(row["rel_type"]).strip()]


def fetch_stats(driver, database: str) -> dict:
    query = """
    MATCH (n:KnowledgeNode)
    WITH collect(n) AS nodes
    CALL (nodes) {
      WITH nodes
      UNWIND nodes AS n
      RETURN count(*) AS node_count,
             count(CASE WHEN trim(coalesce(n.description, '')) = '' THEN 1 END) AS missing_description_count,
             count(CASE WHEN NOT (n)--() THEN 1 END) AS isolated_count
    }
    CALL () {
      MATCH ()-[r]->()
      RETURN count(r) AS relation_count
    }
    RETURN node_count, relation_count, missing_description_count, isolated_count
    """
    record = driver.execute_query(query, database_=database).records[0]
    return {
        "node_count": int(record["node_count"]),
        "relation_count": int(record["relation_count"]),
        "missing_description_count": int(record["missing_description_count"]),
        "isolated_count": int(record["isolated_count"]),
    }


def overlay_curated_nodes(driver, database: str, curated_registry: dict[str, dict]) -> tuple[dict[str, str], dict]:
    existing_nodes = fetch_nodes(driver, database)
    exact_lookup: dict[str, list[KnowledgeNodeRecord]] = defaultdict(list)
    normalized_lookup: dict[str, list[KnowledgeNodeRecord]] = defaultdict(list)
    for node in existing_nodes:
        exact_lookup[node.canonical_name].append(node)
        normalized_lookup[normalize_name(node.canonical_name)].append(node)

    rows: list[dict] = []
    canonical_to_id: dict[str, str] = {}
    inserted = 0
    overlaid = 0

    for canonical_name, item in curated_registry.items():
        candidates = exact_lookup.get(canonical_name) or normalized_lookup.get(normalize_name(canonical_name)) or []
        existing = choose_primary(candidates) if candidates else None
        node_id = existing.node_id if existing else _stable_id("entity", canonical_name)
        if existing:
            overlaid += 1
        else:
            inserted += 1

        legacy_paths = []
        if existing and existing.source_path and not existing.source_path.startswith("curated/"):
            legacy_paths.append(existing.source_path)
        if existing:
            legacy_paths.extend(existing.legacy_source_paths)

        rows.append(
            {
                "node_id": node_id,
                "name": canonical_name,
                "raw_name": canonical_name,
                "canonical_name": canonical_name,
                "entity_type": item["entity_type"],
                "source_structure_id": f"curated::{canonical_name}",
                "source_path": f"curated/{item['source']}",
                "chapter_title": item.get("course_title") or "人工智能课程知识图谱",
                "description": item.get("description", "").strip(),
                "aliases": unique_list((existing.aliases if existing else []) + item.get("aliases", []) + [canonical_name]),
                "facets": [],
                "confidence": 1.0,
                "review_status": "approved",
                "quality_issues": [],
                "source_line_no": 0,
                "source_kind": "curated",
                "quality_tier": "gold",
                "normalized_name": normalize_name(canonical_name),
                "curated_source": item["source"],
                "legacy_source_paths": unique_list(legacy_paths),
                "graph_namespace": "ai_course_kg",
                "data_standard": "industrial_v1",
            }
        )
        canonical_to_id[canonical_name] = node_id

    with driver.session(database=database) as session:
        for start in range(0, len(rows), 200):
            batch = rows[start : start + 200]
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
                  n.normalized_name = coalesce(n.normalized_name, row.normalized_name),
                  n.curated_source = coalesce(n.curated_source, row.curated_source),
                  n.legacy_source_paths = CASE
                    WHEN size(coalesce(n.legacy_source_paths, [])) = 0 THEN row.legacy_source_paths
                    ELSE n.legacy_source_paths
                  END,
                  n.graph_namespace = coalesce(n.graph_namespace, row.graph_namespace),
                  n.data_standard = coalesce(n.data_standard, row.data_standard)
                """,
                rows=batch,
            )

    return canonical_to_id, {"curated_nodes_inserted": inserted, "curated_nodes_overlaid": overlaid}


def merge_curated_relations(driver, database: str, canonical_to_id: dict[str, str], relations: set[tuple[str, str, str]]) -> int:
    rows_by_type: dict[str, list[dict]] = defaultdict(list)
    for source_name, target_name, relation_type in relations:
        source_id = canonical_to_id.get(source_name)
        target_id = canonical_to_id.get(target_name)
        if not source_id or not target_id or source_id == target_id:
            continue
        rows_by_type[relation_type].append(
            {
                "relation_id": _stable_id(relation_type, source_id, target_id),
                "source_id": source_id,
                "target_id": target_id,
                "relation_type": relation_type,
                "confidence": 1.0,
                "source_text": "curated_industrial_registry",
                "source_path": CURATED_RELATION_SOURCE,
                "relation_source": "curated_registry",
            }
        )

    merged = 0
    with driver.session(database=database) as session:
        for relation_type, rows in rows_by_type.items():
            if not rows:
                continue
            merged += len(rows)
            query = f"""
            UNWIND $rows AS row
            MATCH (a:KnowledgeNode {{node_id: row.source_id}})
            MATCH (b:KnowledgeNode {{node_id: row.target_id}})
            MERGE (a)-[r:{relation_type}]->(b)
            SET r += row
            """
            for start in range(0, len(rows), 200):
                session.run(query, rows=rows[start : start + 200])
    return merged


def apply_manual_redirects(driver, database: str) -> list[dict]:
    nodes = fetch_nodes(driver, database)
    by_name = {node.canonical_name: node for node in nodes}
    relation_types = fetch_relation_types(driver, database)
    applied: list[dict] = []

    for source_name, target_name in MANUAL_CANONICAL_REDIRECTS.items():
        loser = by_name.get(source_name)
        winner = by_name.get(target_name)
        if not loser or not winner or loser.node_id == winner.node_id:
            continue
        merge_duplicate_node(driver, database, winner, loser, relation_types)
        applied.append({"from": source_name, "to": target_name})

    return applied


def merge_duplicate_node(driver, database: str, winner: KnowledgeNodeRecord, loser: KnowledgeNodeRecord, relation_types: list[str]) -> None:
    for relation_type in relation_types:
        outgoing_query = f"""
        MATCH (loser:KnowledgeNode {{node_id: $loser_id}})-[r:{relation_type}]->(other:KnowledgeNode)
        RETURN other.node_id AS other_id, properties(r) AS props
        """
        outgoing_rows = driver.execute_query(outgoing_query, {"loser_id": loser.node_id}, database_=database).records
        for row in outgoing_rows:
            other_id = row["other_id"]
            if other_id == winner.node_id:
                continue
            props = dict(row["props"])
            props["relation_id"] = _stable_id(relation_type, winner.node_id, other_id)
            props["source_id"] = winner.node_id
            props["target_id"] = other_id
            upsert_query = f"""
            MATCH (winner:KnowledgeNode {{node_id: $winner_id}})
            MATCH (other:KnowledgeNode {{node_id: $other_id}})
            MERGE (winner)-[r:{relation_type}]->(other)
            SET r += $props
            """
            driver.execute_query(upsert_query, {"winner_id": winner.node_id, "other_id": other_id, "props": props}, database_=database)

        incoming_query = f"""
        MATCH (other:KnowledgeNode)-[r:{relation_type}]->(loser:KnowledgeNode {{node_id: $loser_id}})
        RETURN other.node_id AS other_id, properties(r) AS props
        """
        incoming_rows = driver.execute_query(incoming_query, {"loser_id": loser.node_id}, database_=database).records
        for row in incoming_rows:
            other_id = row["other_id"]
            if other_id == winner.node_id:
                continue
            props = dict(row["props"])
            props["relation_id"] = _stable_id(relation_type, other_id, winner.node_id)
            props["source_id"] = other_id
            props["target_id"] = winner.node_id
            upsert_query = f"""
            MATCH (winner:KnowledgeNode {{node_id: $winner_id}})
            MATCH (other:KnowledgeNode {{node_id: $other_id}})
            MERGE (other)-[r:{relation_type}]->(winner)
            SET r += $props
            """
            driver.execute_query(upsert_query, {"winner_id": winner.node_id, "other_id": other_id, "props": props}, database_=database)

    merged_aliases = unique_list(winner.aliases + loser.aliases + [winner.canonical_name, loser.canonical_name, winner.name, loser.name])
    merged_paths = unique_list(winner.legacy_source_paths + loser.legacy_source_paths + [winner.source_path, loser.source_path])
    driver.execute_query(
        """
        MATCH (winner:KnowledgeNode {node_id: $winner_id})
        SET winner.aliases = $aliases,
            winner.legacy_source_paths = $paths
        """,
        {"winner_id": winner.node_id, "aliases": merged_aliases, "paths": merged_paths},
        database_=database,
    )
    driver.execute_query(
        """
        MATCH (loser:KnowledgeNode {node_id: $loser_id})
        DETACH DELETE loser
        """,
        {"loser_id": loser.node_id},
        database_=database,
    )


def dedupe_near_duplicates(driver, database: str) -> list[dict]:
    nodes = fetch_nodes(driver, database)
    relation_types = fetch_relation_types(driver, database)
    groups: dict[str, list[KnowledgeNodeRecord]] = defaultdict(list)
    for node in nodes:
        groups[normalize_name(node.canonical_name)].append(node)

    merged_groups: list[dict] = []
    for normalized_name, items in groups.items():
        if len(items) <= 1:
            continue
        winner = choose_primary(items)
        losers = [item for item in items if item.node_id != winner.node_id]
        for loser in losers:
            merge_duplicate_node(driver, database, winner, loser, relation_types)
        merged_groups.append(
            {
                "normalized_name": normalized_name,
                "winner": winner.canonical_name,
                "merged": [item.canonical_name for item in losers],
            }
        )
    return merged_groups


def is_suspicious(node: KnowledgeNodeRecord) -> bool:
    if node.source_path.startswith("curated/"):
        return False
    if any(node.canonical_name.startswith(prefix) for prefix in SUSPICIOUS_PREFIXES):
        return True
    return any(token in node.canonical_name for token in SUSPICIOUS_TOKENS)


def delete_suspicious_nodes(driver, database: str) -> list[str]:
    nodes = fetch_nodes(driver, database)
    candidates = [
        node for node in nodes
        if is_suspicious(node) and (node.degree <= 2 or not node.description.strip())
    ]
    deleted = [node.canonical_name for node in candidates]
    for node in candidates:
        driver.execute_query(
            """
            MATCH (n:KnowledgeNode {node_id: $node_id})
            DETACH DELETE n
            """,
            {"node_id": node.node_id},
            database_=database,
        )
    return deleted


def infer_parent_name(node: KnowledgeNodeRecord) -> str:
    manual = MANUAL_PARENT_MAP.get(node.canonical_name)
    if manual:
        return manual
    text = f"{node.canonical_name} {node.source_path}"
    for keyword, parent in SOURCE_PARENT_RULES:
        if keyword in text:
            return parent
    return ""


def add_inferred_parent_relations(driver, database: str) -> list[dict]:
    nodes = fetch_nodes(driver, database)
    by_name = {node.canonical_name: node for node in nodes}
    added: list[dict] = []

    for node in nodes:
        if node.source_path.startswith("curated/") or node.degree != 0:
            continue
        parent_name = infer_parent_name(node)
        parent = by_name.get(parent_name)
        if not parent or parent.node_id == node.node_id:
            continue
        relation = {
            "relation_id": _stable_id("SUBTOPIC_OF", node.node_id, parent.node_id),
            "source_id": node.node_id,
            "target_id": parent.node_id,
            "relation_type": "SUBTOPIC_OF",
            "confidence": 0.92,
            "source_text": "industrial_normalization_parent_inference",
            "source_path": NORMALIZATION_SOURCE,
            "relation_source": "manual_rule",
        }
        driver.execute_query(
            """
            MATCH (child:KnowledgeNode {node_id: $source_id})
            MATCH (parent:KnowledgeNode {node_id: $target_id})
            MERGE (child)-[r:SUBTOPIC_OF]->(parent)
            SET r += $row
            """,
            {"source_id": node.node_id, "target_id": parent.node_id, "row": relation},
            database_=database,
        )
        added.append({"child": node.canonical_name, "parent": parent.canonical_name, "relation_type": "SUBTOPIC_OF"})
    return added


def backfill_descriptions(driver, database: str) -> list[str]:
    updated: list[str] = []
    for canonical_name, description in DESCRIPTION_BACKFILL_MAP.items():
        records = driver.execute_query(
            """
            MATCH (n:KnowledgeNode {canonical_name: $canonical_name})
            WHERE trim(coalesce(n.description, '')) = ''
            SET n.description = $description
            RETURN n.canonical_name AS name
            """,
            {"canonical_name": canonical_name, "description": description},
            database_=database,
        ).records
        if records:
            updated.append(canonical_name)
    return updated


def match_taxonomy(texts: list[str], topic_to_domain: dict[str, str], ordered_terms: list[str], domain_names: set[str]) -> tuple[str, str]:
    joined = " ".join(text for text in texts if text).strip()
    for term in ordered_terms:
        if term and term in joined:
            domain = topic_to_domain[term]
            topic = term if term not in domain_names else ""
            return domain, topic
    return "", ""


def apply_metadata(driver, database: str, topic_to_domain: dict[str, str], ordered_terms: list[str], domain_names: set[str]) -> int:
    timestamp = datetime.now(timezone.utc).isoformat()
    nodes = fetch_nodes(driver, database)
    rows: list[dict] = []

    for node in nodes:
        texts = [node.canonical_name, *node.aliases, node.source_path]
        taxonomy_domain, taxonomy_topic = match_taxonomy(texts, topic_to_domain, ordered_terms, domain_names)
        source_kind = "curated" if node.source_path.startswith("curated/") else "extracted"
        quality_tier = "gold" if source_kind == "curated" else ("silver" if node.description.strip() and node.degree > 0 else "bronze")
        rows.append(
            {
                "node_id": node.node_id,
                "normalized_name": normalize_name(node.canonical_name),
                "source_kind": source_kind,
                "quality_tier": quality_tier,
                "taxonomy_domain": taxonomy_domain,
                "taxonomy_topic": taxonomy_topic,
                "graph_namespace": "ai_course_kg",
                "data_standard": "industrial_v1",
                "normalized_at": timestamp,
            }
        )

    with driver.session(database=database) as session:
        for start in range(0, len(rows), 200):
            session.run(
                """
                UNWIND $rows AS row
                MATCH (n:KnowledgeNode {node_id: row.node_id})
                SET n += row
                """,
                rows=rows[start : start + 200],
            )
    return len(rows)


def create_graph_indexes(driver) -> None:
    create_constraints(driver)
    with driver.session() as session:
        for statement in GRAPH_INDEX_STATEMENTS:
            session.run(statement)


def write_report(report: dict) -> Path:
    REPORT_DIR.mkdir(parents=True, exist_ok=True)
    path = REPORT_DIR / "normalization_report.json"
    path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    return path


def normalize_graph(uri: str, username: str, password: str, database: str = "neo4j") -> dict:
    curated_registry, curated_relations = load_curated_registry()
    topic_to_domain, ordered_terms, domain_names = load_reference_taxonomy()
    driver = GraphDatabase.driver(uri, auth=(username, password))
    try:
        before = fetch_stats(driver, database)
        canonical_to_id, curated_report = overlay_curated_nodes(driver, database, curated_registry)
        merged_relation_count = merge_curated_relations(driver, database, canonical_to_id, curated_relations)
        manual_redirects = apply_manual_redirects(driver, database)
        merged_duplicates = dedupe_near_duplicates(driver, database)
        deleted_nodes = delete_suspicious_nodes(driver, database)
        inferred_relations = add_inferred_parent_relations(driver, database)
        backfilled_descriptions = backfill_descriptions(driver, database)
        metadata_count = apply_metadata(driver, database, topic_to_domain, ordered_terms, domain_names)
        create_graph_indexes(driver)
        after = fetch_stats(driver, database)
        if after["node_count"] < 700:
            raise ValueError(f"normalized graph has only {after['node_count']} knowledge nodes, below the 700 requirement")
        report = {
            "before": before,
            "after": after,
            "curated_report": {
                **curated_report,
                "curated_relations_merged": merged_relation_count,
                "curated_registry_node_count": len(curated_registry),
                "curated_registry_relation_count": len(curated_relations),
            },
            "merged_duplicates": merged_duplicates,
            "manual_redirects": manual_redirects,
            "deleted_suspicious_nodes": deleted_nodes,
            "inferred_relations": inferred_relations,
            "backfilled_descriptions": backfilled_descriptions,
            "metadata_rows_updated": metadata_count,
        }
        report["report_path"] = str(write_report(report))
        return report
    finally:
        driver.close()


def main() -> None:
    load_dotenv(Path(__file__).resolve().parents[4] / ".env")
    parser = build_parser()
    args = parser.parse_args()
    report = normalize_graph(
        uri=args.neo4j_uri,
        username=args.neo4j_user,
        password=args.neo4j_password,
        database=args.neo4j_database,
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
