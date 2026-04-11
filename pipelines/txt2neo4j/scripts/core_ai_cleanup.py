"""Delete legacy extracted nodes and re-merge the curated core AI ontology.

Run from `pipelines/txt2neo4j` with `PYTHONPATH=src`.
"""
from __future__ import annotations

import argparse
import json
import os
from datetime import datetime, timezone
from pathlib import Path

from dotenv import load_dotenv
from neo4j import GraphDatabase

from txt2neo4j.curated_graph import merge_curated_graph

REPORT_DIR = Path(__file__).resolve().parents[1] / "artifacts" / "core_ai_cleanup"
CORE_CHECK_NAMES = [
    "人工智能",
    "数学基础",
    "线性代数",
    "微积分",
    "概率统计",
    "最优化",
    "机器学习",
    "监督学习",
    "非监督学习",
    "强化学习",
    "深度学习",
    "神经网络",
    "多层感知机",
    "损失函数",
    "梯度下降",
    "反向传播",
    "正则化",
    "模型评估",
    "卷积神经网络",
    "循环神经网络",
    "注意力机制",
    "Transformer",
    "大语言模型",
    "检索增强生成",
    "知识图谱",
    "智能体",
]


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Clean legacy Neo4j textbook nodes and merge the curated core AI seed."
    )
    parser.add_argument("--neo4j-uri", default=os.getenv("NEO4J_URI", "bolt://127.0.0.1:7687"))
    parser.add_argument("--neo4j-user", default=os.getenv("NEO4J_USER", os.getenv("NEO4J_USERNAME", "neo4j")))
    parser.add_argument("--neo4j-password", default=os.getenv("NEO4J_PASSWORD", ""))
    parser.add_argument("--neo4j-database", default=os.getenv("NEO4J_DATABASE", "neo4j"))
    parser.add_argument("--dry-run", action="store_true")
    return parser


def fetch_summary(driver, database: str) -> dict:
    query = """
    MATCH (n:KnowledgeNode)
    WITH collect(n) AS nodes
    CALL (nodes) {
      WITH nodes
      UNWIND nodes AS n
      RETURN count(*) AS node_count,
             count(CASE WHEN coalesce(n.source_path, '') STARTS WITH 'curated/' THEN 1 END) AS curated_node_count,
             count(CASE WHEN NOT coalesce(n.source_path, '') STARTS WITH 'curated/' THEN 1 END) AS legacy_node_count
    }
    CALL () {
      MATCH ()-[r]->()
      RETURN count(r) AS relation_count
    }
    RETURN node_count, curated_node_count, legacy_node_count, relation_count
    """
    record = driver.execute_query(query, database_=database).records[0]
    return {
        "node_count": int(record["node_count"]),
        "curated_node_count": int(record["curated_node_count"]),
        "legacy_node_count": int(record["legacy_node_count"]),
        "relation_count": int(record["relation_count"]),
    }


def fetch_legacy_samples(driver, database: str, limit: int = 40) -> list[dict]:
    query = """
    MATCH (n:KnowledgeNode)
    WHERE NOT coalesce(n.source_path, '') STARTS WITH 'curated/'
    RETURN n.canonical_name AS name, n.source_path AS source_path
    ORDER BY n.canonical_name
    LIMIT $limit
    """
    rows = driver.execute_query(query, {"limit": limit}, database_=database).records
    return [{"name": row["name"], "source_path": row["source_path"]} for row in rows]


def fetch_core_presence(driver, database: str) -> list[str]:
    query = """
    MATCH (n:KnowledgeNode)
    WHERE n.canonical_name IN $names
    RETURN n.canonical_name AS name
    ORDER BY name
    """
    rows = driver.execute_query(query, {"names": CORE_CHECK_NAMES}, database_=database).records
    return [str(row["name"]) for row in rows]


def fetch_legacy_names(driver, database: str) -> list[str]:
    query = """
    MATCH (n:KnowledgeNode)
    WHERE NOT coalesce(n.source_path, '') STARTS WITH 'curated/'
    RETURN n.canonical_name AS name
    ORDER BY n.canonical_name
    """
    rows = driver.execute_query(query, database_=database).records
    return [str(row["name"]) for row in rows]


def delete_legacy_nodes(driver, database: str) -> int:
    query = """
    MATCH (n:KnowledgeNode)
    WHERE NOT coalesce(n.source_path, '') STARTS WITH 'curated/'
    DETACH DELETE n
    """
    driver.execute_query(query, database_=database)
    return 0


def fetch_conflicting_part_of_relations(driver, database: str) -> list[dict]:
    query = """
    MATCH (a:KnowledgeNode)-[:PART_OF]->(b:KnowledgeNode)
    WHERE EXISTS { MATCH (a)-[:SUBTOPIC_OF]->(b) }
    RETURN a.canonical_name AS source, b.canonical_name AS target
    ORDER BY source, target
    """
    rows = driver.execute_query(query, database_=database).records
    return [{"source": row["source"], "target": row["target"]} for row in rows]


def delete_conflicting_part_of_relations(driver, database: str) -> None:
    query = """
    MATCH (a:KnowledgeNode)-[r:PART_OF]->(b:KnowledgeNode)
    WHERE EXISTS { MATCH (a)-[:SUBTOPIC_OF]->(b) }
    DELETE r
    """
    driver.execute_query(query, database_=database)


def write_report(report: dict) -> str:
    REPORT_DIR.mkdir(parents=True, exist_ok=True)
    path = REPORT_DIR / "cleanup_report.json"
    path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    return str(path)


def main() -> None:
    root = Path(__file__).resolve().parents[1]
    load_dotenv(root.parents[1] / ".env")
    parser = build_parser()
    args = parser.parse_args()

    driver = GraphDatabase.driver(args.neo4j_uri, auth=(args.neo4j_user, args.neo4j_password))
    try:
        before = fetch_summary(driver, args.neo4j_database)
        legacy_samples = fetch_legacy_samples(driver, args.neo4j_database)
        legacy_names = fetch_legacy_names(driver, args.neo4j_database)
        conflicting_part_of = fetch_conflicting_part_of_relations(driver, args.neo4j_database)
        core_before = fetch_core_presence(driver, args.neo4j_database)

        merge_report: dict = {}
        deleted_count = len(legacy_names)

        if not args.dry_run:
            delete_legacy_nodes(driver, args.neo4j_database)
            delete_conflicting_part_of_relations(driver, args.neo4j_database)
    finally:
        driver.close()

    if not args.dry_run:
        merge_report = merge_curated_graph(
            uri=args.neo4j_uri,
            username=args.neo4j_user,
            password=args.neo4j_password,
            database=args.neo4j_database,
        )

    driver = GraphDatabase.driver(args.neo4j_uri, auth=(args.neo4j_user, args.neo4j_password))
    try:
        after = fetch_summary(driver, args.neo4j_database)
        core_after = fetch_core_presence(driver, args.neo4j_database)
    finally:
        driver.close()

    report = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "dry_run": args.dry_run,
        "before": before,
        "after": after,
        "legacy_node_samples": legacy_samples,
        "legacy_nodes_deleted": legacy_names if not args.dry_run else [],
        "legacy_node_delete_count": deleted_count,
        "conflicting_part_of_relations_deleted": conflicting_part_of if not args.dry_run else [],
        "core_presence_before": core_before,
        "core_presence_after": core_after,
        "merge_report": merge_report,
    }
    report["report_path"] = write_report(report)
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
