"""One-off / ad-hoc Neo4j graph naming & quality audit. Run from repo with PYTHONPATH=src."""
from __future__ import annotations

import json
import os
from pathlib import Path

from dotenv import load_dotenv
from neo4j import GraphDatabase

from txt2neo4j.neo4j_import import SUPPORTED_RELATION_TYPES


def main() -> None:
    root = Path(__file__).resolve().parents[1]
    load_dotenv(root.parents[1] / ".env")
    uri = os.getenv("NEO4J_URI")
    user = os.getenv("NEO4J_USER")
    pw = os.getenv("NEO4J_PASSWORD")
    database = os.getenv("NEO4J_DATABASE", "neo4j")
    driver = GraphDatabase.driver(uri, auth=(user, pw))
    report: dict = {}

    try:
        with driver.session(database=database) as s:
            report["totals"] = {
                "nodes": s.run("MATCH (n:KnowledgeNode) RETURN count(n) AS c").single()["c"],
                "relationships": s.run("MATCH ()-[r]->() RETURN count(r) AS c").single()["c"],
            }
            report["duplicate_name_groups"] = s.run(
                "MATCH (n:KnowledgeNode) WITH n.name AS k, count(*) AS c "
                "WHERE k IS NOT NULL AND c > 1 RETURN count(*) AS c"
            ).single()["c"]
            report["duplicate_canonical_groups"] = s.run(
                "MATCH (n:KnowledgeNode) WITH n.canonical_name AS k, count(*) AS c "
                "WHERE k IS NOT NULL AND c > 1 RETURN count(*) AS c"
            ).single()["c"]
            report["nodes_empty_description"] = s.run(
                "MATCH (n:KnowledgeNode) WHERE n.description IS NULL OR trim(toString(n.description)) = '' "
                "RETURN count(n) AS c"
            ).single()["c"]
            report["curated_empty_description"] = s.run(
                "MATCH (n:KnowledgeNode) WHERE n.source_path STARTS WITH 'curated' "
                "AND (n.description IS NULL OR trim(toString(n.description)) = '') RETURN count(n) AS c"
            ).single()["c"]
            report["name_canonical_mismatch"] = s.run(
                "MATCH (n:KnowledgeNode) WHERE n.name IS NOT NULL AND n.canonical_name IS NOT NULL "
                "AND trim(toString(n.name)) <> trim(toString(n.canonical_name)) RETURN count(n) AS c"
            ).single()["c"]
            report["whitespace_in_name_fields"] = s.run(
                "MATCH (n:KnowledgeNode) WHERE toString(n.name) <> trim(toString(n.name)) "
                "OR toString(n.canonical_name) <> trim(toString(n.canonical_name)) RETURN count(n) AS c"
            ).single()["c"]
            report["isolated_knowledge_nodes"] = s.run(
                "MATCH (n:KnowledgeNode) WHERE NOT (n)--() RETURN count(n) AS c"
            ).single()["c"]
            report["curated_isolated"] = s.run(
                "MATCH (n:KnowledgeNode) WHERE n.source_path STARTS WITH 'curated' "
                "AND NOT (n)--() RETURN count(n) AS c"
            ).single()["c"]
            report["duplicate_node_id_groups"] = s.run(
                "MATCH (n:KnowledgeNode) WITH n.node_id AS id, count(*) AS c WHERE c > 1 RETURN count(*) AS c"
            ).single()["c"]
            rows = s.run(
                "MATCH ()-[r]->() RETURN type(r) AS t, count(*) AS c ORDER BY c DESC"
            )
            rel_counts = {row["t"]: row["c"] for row in rows}
            report["relationship_types"] = rel_counts
            unsupported = [t for t in rel_counts if t not in SUPPORTED_RELATION_TYPES]
            report["unsupported_relationship_types"] = unsupported
            report["nodes_with_uFFFD"] = s.run(
                r"MATCH (n:KnowledgeNode) WHERE toString(coalesce(n.description,'')) CONTAINS '\uFFFD' "
                r"OR toString(coalesce(n.name,'')) CONTAINS '\uFFFD' RETURN count(n) AS c"
            ).single()["c"]

            report["samples_empty_description"] = [
                {"canonical_name": row["can"], "source_path": row["src"]}
                for row in s.run(
                    "MATCH (n:KnowledgeNode) WHERE n.description IS NULL OR trim(toString(n.description)) = '' "
                    "RETURN n.canonical_name AS can, n.source_path AS src ORDER BY src LIMIT 20"
                )
            ]
            report["samples_isolated"] = [
                {"canonical_name": row["can"], "source_path": row["src"]}
                for row in s.run(
                    "MATCH (n:KnowledgeNode) WHERE NOT (n)--() "
                    "RETURN n.canonical_name AS can, n.source_path AS src ORDER BY src LIMIT 25"
                )
            ]
    finally:
        driver.close()

    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
