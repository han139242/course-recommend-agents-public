from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

from dotenv import load_dotenv

from .curated_graph import merge_curated_graph
from .neo4j_import import import_bundle
from .pipeline import run_pipeline


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="txt2neo4j",
        description="High-precision TXT to Neo4j extraction pipeline",
    )
    parser.add_argument("--input", required=True, help="Path to the source TXT file")
    parser.add_argument(
        "--output-dir",
        default="artifacts/txt2neo4j",
        help="Directory for cleaned text, extracted nodes, relations, and CSV exports",
    )
    parser.add_argument(
        "--import-neo4j",
        action="store_true",
        help="Import exported graph data into Neo4j after extraction",
    )
    parser.add_argument(
        "--import-curated-seed",
        action="store_true",
        help="Merge a curated AI foundation taxonomy into Neo4j without creating duplicates",
    )
    parser.add_argument(
        "--curated-seed-name",
        default="",
        help="Curated seed filename or batch name under config/curated_batches",
    )
    parser.add_argument("--neo4j-uri", default=os.getenv("NEO4J_URI", "bolt://127.0.0.1:7687"))
    parser.add_argument("--neo4j-user", default=os.getenv("NEO4J_USER", "neo4j"))
    parser.add_argument("--neo4j-password", default=os.getenv("NEO4J_PASSWORD", ""))
    parser.add_argument("--neo4j-database", default=os.getenv("NEO4J_DATABASE", "neo4j"))
    return parser


def main() -> None:
    load_dotenv(Path(__file__).resolve().parents[4] / ".env")
    parser = build_parser()
    args = parser.parse_args()

    input_path = Path(args.input).resolve()
    output_dir = Path(args.output_dir).resolve()

    bundle = run_pipeline(input_path, output_dir)

    if args.import_neo4j:
        import_bundle(
            bundle=bundle,
            uri=args.neo4j_uri,
            username=args.neo4j_user,
            password=args.neo4j_password,
            database=args.neo4j_database,
        )

    curated_report = None
    if args.import_curated_seed:
        curated_report = merge_curated_graph(
            uri=args.neo4j_uri,
            username=args.neo4j_user,
            password=args.neo4j_password,
            database=args.neo4j_database,
            seed_name_or_path=args.curated_seed_name or None,
        )

    print(
        json.dumps(
            {
                "source": str(input_path),
                "output_dir": str(output_dir),
                "structure_nodes": len(bundle.structure_nodes),
                "knowledge_nodes": len(bundle.knowledge_nodes),
                "relations": len(bundle.relations),
                "quality_gate_passed": bundle.quality_report["quality_gate_passed"],
                "curated_report": curated_report,
            },
            ensure_ascii=False,
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
