from __future__ import annotations

import argparse
import json
from pathlib import Path

from .neo4j_import import SUPPORTED_RELATION_TYPES

PIPELINE_ROOT = Path(__file__).resolve().parents[2]
CURATED_SEED_PATH = PIPELINE_ROOT / "config" / "curated_seed_graph.json"
CURATED_BATCH_DIR = PIPELINE_ROOT / "config" / "curated_batches"
VALID_ENTITY_TYPES = {"topic", "concept", "method", "algorithm", "application", "exercise"}


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


def iter_registry_paths(target_path: Path) -> list[Path]:
    paths = [CURATED_SEED_PATH, *sorted(CURATED_BATCH_DIR.glob("*.json"))]
    unique_paths: list[Path] = []
    seen: set[Path] = set()
    target_resolved = target_path.resolve()
    for path in paths:
        resolved = path.resolve()
        if resolved == target_resolved or resolved in seen or not path.exists():
            continue
        seen.add(resolved)
        unique_paths.append(path)
    return unique_paths


def validate_seed_path(seed_path: Path) -> dict:
    payload = load_json_file(seed_path)
    nodes = payload.get("nodes", [])
    relations = payload.get("relations", [])

    errors: list[str] = []
    warnings: list[str] = []

    if not payload.get("source"):
        errors.append("missing source field")

    current_names: list[str] = []
    current_name_set: set[str] = set()
    duplicate_names_in_batch: set[str] = set()
    invalid_entity_types: list[str] = []
    empty_descriptions: list[str] = []

    for node in nodes:
        canonical_name = str(node.get("canonical_name", "")).strip()
        entity_type = str(node.get("entity_type", "")).strip()
        description = str(node.get("description", "")).strip()

        if not canonical_name:
            errors.append("node with empty canonical_name")
            continue
        if canonical_name in current_name_set:
            duplicate_names_in_batch.add(canonical_name)
        current_name_set.add(canonical_name)
        current_names.append(canonical_name)

        if entity_type not in VALID_ENTITY_TYPES:
            invalid_entity_types.append(f"{canonical_name}: {entity_type}")
        if not description:
            empty_descriptions.append(canonical_name)

    if duplicate_names_in_batch:
        errors.append(
            "duplicate canonical_name in batch: " + ", ".join(sorted(duplicate_names_in_batch))
        )
    if invalid_entity_types:
        errors.append("invalid entity_type: " + "; ".join(sorted(invalid_entity_types)))
    if empty_descriptions:
        errors.append("nodes missing description: " + ", ".join(sorted(empty_descriptions)))

    relation_tuples: list[tuple[str, str, str]] = []
    relation_set: set[tuple[str, str, str]] = set()
    duplicate_relations_in_batch: set[tuple[str, str, str]] = set()
    invalid_relation_types: list[str] = []
    self_loops: list[str] = []

    for relation in relations:
        if len(relation) != 3:
            errors.append(f"invalid relation record shape: {relation}")
            continue
        source_name, target_name, relation_type = relation
        key = (source_name, target_name, relation_type)
        relation_tuples.append(key)
        if key in relation_set:
            duplicate_relations_in_batch.add(key)
        relation_set.add(key)

        if relation_type not in SUPPORTED_RELATION_TYPES:
            invalid_relation_types.append(f"{source_name}->{target_name}: {relation_type}")
        if source_name == target_name:
            self_loops.append(f"{source_name}->{target_name}: {relation_type}")

    if duplicate_relations_in_batch:
        errors.append(
            "duplicate relations in batch: "
            + "; ".join(f"{s}->{t}:{r}" for s, t, r in sorted(duplicate_relations_in_batch))
        )
    if invalid_relation_types:
        errors.append("invalid relation types: " + "; ".join(sorted(invalid_relation_types)))
    if self_loops:
        errors.append("self loop relations: " + "; ".join(sorted(self_loops)))

    external_name_to_source: dict[str, str] = {}
    external_relations: set[tuple[str, str, str]] = set()
    for path in iter_registry_paths(seed_path):
        registry_payload = load_json_file(path)
        for node in registry_payload.get("nodes", []):
            canonical_name = str(node.get("canonical_name", "")).strip()
            if canonical_name and canonical_name not in external_name_to_source:
                external_name_to_source[canonical_name] = path.name
        for relation in registry_payload.get("relations", []):
            if len(relation) == 3:
                external_relations.add(tuple(relation))

    cross_file_duplicate_nodes = [
        f"{name} @ {external_name_to_source[name]}"
        for name in sorted(current_name_set)
        if name in external_name_to_source
    ]
    if cross_file_duplicate_nodes:
        errors.append(
            "canonical_name already defined in other curated files: "
            + "; ".join(cross_file_duplicate_nodes)
        )

    available_names = current_name_set | set(external_name_to_source)
    missing_relation_endpoints: list[str] = []
    for source_name, target_name, relation_type in relation_tuples:
        if source_name not in available_names:
            missing_relation_endpoints.append(f"missing source {source_name} for {relation_type}")
        if target_name not in available_names:
            missing_relation_endpoints.append(f"missing target {target_name} for {relation_type}")
    if missing_relation_endpoints:
        errors.append("relations reference undefined nodes: " + "; ".join(sorted(missing_relation_endpoints)))

    repeated_relations = [
        f"{s}->{t}:{r}" for s, t, r in sorted(relation_set) if (s, t, r) in external_relations
    ]
    if repeated_relations:
        warnings.append(
            "relations already exist in other curated files: " + "; ".join(repeated_relations)
        )

    return {
        "valid": not errors,
        "source": payload.get("source", ""),
        "node_count": len(nodes),
        "relation_count": len(relations),
        "errors": errors,
        "warnings": warnings,
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="Validate curated seed or batch before import.")
    parser.add_argument("--seed-name", default="")
    args = parser.parse_args()
    result = validate_seed_path(resolve_seed_path(args.seed_name or None))
    print(json.dumps(result, ensure_ascii=False, indent=2))
    if not result["valid"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
