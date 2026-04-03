from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

from dotenv import load_dotenv
from neo4j import GraphDatabase


def build_queries(batch_source: str | None) -> dict[str, str]:
    queries = {
        "total_nodes": "MATCH (n) RETURN count(n) AS c",
        "total_relationships": "MATCH ()-[r]->() RETURN count(r) AS c",
        "duplicate_name_groups": (
            "MATCH (n:KnowledgeNode) "
            "WITH n.name AS key, count(*) AS c "
            "WHERE key IS NOT NULL AND c > 1 "
            "RETURN count(*) AS c"
        ),
        "duplicate_canonical_groups": (
            "MATCH (n:KnowledgeNode) "
            "WITH n.canonical_name AS key, count(*) AS c "
            "WHERE key IS NOT NULL AND c > 1 "
            "RETURN count(*) AS c"
        ),
    }
    if batch_source:
        queries["batch_nodes"] = (
            "MATCH (n) "
            f"WHERE n.source_path STARTS WITH 'curated/{batch_source}' "
            "RETURN count(n) AS c"
        )
        queries["batch_relationships"] = (
            "MATCH ()-[r]->() "
            f"WHERE r.source_path STARTS WITH 'curated/{batch_source}' "
            "RETURN count(r) AS c"
        )
    return queries


def main() -> None:
    parser = argparse.ArgumentParser(description="Verify Neo4j graph state.")
    parser.add_argument("--env-file", default=str(Path(__file__).resolve().parents[4] / ".env"))
    parser.add_argument("--batch-source", default=None)
    args = parser.parse_args()

    load_dotenv(args.env_file)

    driver = GraphDatabase.driver(
        os.getenv("NEO4J_URI"),
        auth=(os.getenv("NEO4J_USER"), os.getenv("NEO4J_PASSWORD")),
    )

    try:
        results: dict[str, int] = {}
        with driver.session() as session:
            for key, query in build_queries(args.batch_source).items():
                results[key] = session.run(query).single()["c"]
        print(json.dumps(results, ensure_ascii=False, indent=2))
    finally:
        driver.close()


if __name__ == "__main__":
    main()
