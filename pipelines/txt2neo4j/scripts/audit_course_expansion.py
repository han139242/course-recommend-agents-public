from __future__ import annotations

import json
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[1]
SRC_ROOT = PROJECT_ROOT / "src"

if str(SRC_ROOT) not in sys.path:
    sys.path.insert(0, str(SRC_ROOT))

from txt2neo4j.curated_graph import resolve_seed_paths_from_pattern  # noqa: E402
from txt2neo4j.validate_curated_batch import validate_seed_path  # noqa: E402

REPORT_DIR = PROJECT_ROOT / "artifacts" / "course_expansion"
REPORT_PATH = REPORT_DIR / "course_expansion_audit.json"


def main() -> None:
    seed_paths = resolve_seed_paths_from_pattern("course_*.json")
    results: list[dict] = []

    for path in seed_paths:
        report = validate_seed_path(path)
        results.append(
            {
                "file": path.name,
                "source": report["source"],
                "valid": report["valid"],
                "node_count": report["node_count"],
                "relation_count": report["relation_count"],
                "relation_density": report["metrics"]["relation_density"],
                "reusable_reference_count": report["metrics"]["reusable_reference_count"],
                "isolated_node_count": report["metrics"]["isolated_node_count"],
                "target_node_range": report["metrics"]["target_node_range"],
                "warning_count": len(report["warnings"]),
                "error_count": len(report["errors"]),
                "warnings": report["warnings"],
                "errors": report["errors"],
            }
        )

    valid_count = sum(1 for item in results if item["valid"])
    total_nodes = sum(int(item["node_count"]) for item in results)
    total_relations = sum(int(item["relation_count"]) for item in results)
    avg_density = round(
        sum(float(item["relation_density"]) for item in results) / len(results),
        4,
    ) if results else 0.0

    payload = {
        "course_count": len(results),
        "valid_course_count": valid_count,
        "invalid_course_count": len(results) - valid_count,
        "total_nodes": total_nodes,
        "total_relations": total_relations,
        "average_relation_density": avg_density,
        "all_valid": valid_count == len(results),
        "courses": results,
    }

    REPORT_DIR.mkdir(parents=True, exist_ok=True)
    REPORT_PATH.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({**payload, "report_path": str(REPORT_PATH)}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
