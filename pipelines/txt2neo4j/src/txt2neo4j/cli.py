from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

from dotenv import load_dotenv

from .curated_graph import merge_curated_graph
from .frontier_pipeline import run_frontier_pipeline
from .neo4j_import import import_bundle, import_frontier_bundle
from .pipeline import run_pipeline


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="txt2neo4j",
        description="High-precision TXT to Neo4j extraction pipeline",
    )
    parser.add_argument("--input", help="Path to the source TXT file")
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
        "--frontier-input",
        help="Path to local frontier paper metadata in JSON or JSONL format",
    )
    parser.add_argument(
        "--frontier-output-dir",
        default="artifacts/frontier",
        help="Directory for frontier digests, topic mappings, and Neo4j CSV exports",
    )
    parser.add_argument(
        "--import-frontier-neo4j",
        action="store_true",
        help="Import generated frontier metadata into Neo4j after extraction",
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

    if not args.input and not args.frontier_input and not args.import_curated_seed:
        parser.error("At least one of --input, --frontier-input, or --import-curated-seed is required.")

    bundle = None
    frontier_bundle = None
    input_path = Path(args.input).resolve() if args.input else None
    output_dir = Path(args.output_dir).resolve()
    frontier_input_path = Path(args.frontier_input).resolve() if args.frontier_input else None
    frontier_output_dir = Path(args.frontier_output_dir).resolve()

    if input_path:
        bundle = run_pipeline(input_path, output_dir)

        if args.import_neo4j:
            import_bundle(
                bundle=bundle,
                uri=args.neo4j_uri,
                username=args.neo4j_user,
                password=args.neo4j_password,
                database=args.neo4j_database,
            )

    if frontier_input_path:
        frontier_bundle = run_frontier_pipeline(frontier_input_path, frontier_output_dir)

        if args.import_frontier_neo4j or args.import_neo4j:
            import_frontier_bundle(
                bundle=frontier_bundle,
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
                "source": str(input_path) if input_path else "",
                "output_dir": str(output_dir),
                "structure_nodes": len(bundle.structure_nodes) if bundle else 0,
                "knowledge_nodes": len(bundle.knowledge_nodes) if bundle else 0,
                "relations": len(bundle.relations) if bundle else 0,
                "quality_gate_passed": bundle.quality_report["quality_gate_passed"] if bundle else None,
                "frontier_source": str(frontier_input_path) if frontier_input_path else "",
                "frontier_output_dir": str(frontier_output_dir),
                "frontier_papers": len(frontier_bundle.papers) if frontier_bundle else 0,
                "frontier_topics": len(frontier_bundle.topics) if frontier_bundle else 0,
                "frontier_digests": len(frontier_bundle.digests) if frontier_bundle else 0,
                "curated_report": curated_report,
            },
            ensure_ascii=False,
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
