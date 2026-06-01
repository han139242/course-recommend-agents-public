from __future__ import annotations

import argparse
import json
from pathlib import Path

from .models import COURSE_ENTITY_TYPE_VALUES, ENTITY_TYPE_VALUES
from .neo4j_import import SUPPORTED_RELATION_TYPES

PIPELINE_ROOT = Path(__file__).resolve().parents[2]
CURATED_SEED_PATH = PIPELINE_ROOT / "config" / "curated_seed_graph.json"
CURATED_BATCH_DIR = PIPELINE_ROOT / "config" / "curated_batches"
COURSE_EXPANSION_PLAN_PATH = PIPELINE_ROOT / "config" / "course_expansion_plan.json"
VALID_ENTITY_TYPES = set(ENTITY_TYPE_VALUES)
COURSE_PLAN_ENTITY_TYPES = set(COURSE_ENTITY_TYPE_VALUES)
RELATION_DENSITY_RANGE = (1.5, 2.1)


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


def load_external_node_registry(target_path: Path) -> dict[str, dict]:
    registry: dict[str, dict] = {}
    for path in iter_registry_paths(target_path):
        payload = load_json_file(path)
        for node in payload.get("nodes", []):
            canonical_name = str(node.get("canonical_name", "")).strip()
            if not canonical_name or canonical_name in registry:
                continue
            registry[canonical_name] = {
                "path": path.name,
                "source": str(payload.get("source", "")).strip(),
                "course_title": str(payload.get("course_title", "")).strip(),
                "entity_type": str(node.get("entity_type", "")).strip(),
                "description": str(node.get("description", "")).strip(),
            }
    return registry


def load_course_plan_index() -> dict[str, dict]:
    if not COURSE_EXPANSION_PLAN_PATH.exists():
        return {}
    payload = load_json_file(COURSE_EXPANSION_PLAN_PATH)
    index: dict[str, dict] = {}
    for course in payload.get("courses", []):
        source = str(course.get("source", "")).strip()
        if source and source not in index:
            index[source] = course
    return index


def validate_seed_path(seed_path: Path) -> dict:
    payload = load_json_file(seed_path)
    source = str(payload.get("source", "")).strip()
    course_title = str(payload.get("course_title", "")).strip()
    nodes = payload.get("nodes", [])
    relations = payload.get("relations", [])

    errors: list[str] = []
    warnings: list[str] = []
    course_plan_index = load_course_plan_index()
    course_plan = course_plan_index.get(source)
    is_plan_managed_course = bool(course_plan) or source.startswith("course_")
    allowed_entity_types = COURSE_PLAN_ENTITY_TYPES if is_plan_managed_course else VALID_ENTITY_TYPES

    if not source:
        errors.append("missing source field")
    elif source.startswith("course_") and not course_plan:
        errors.append(f"source {source} is not registered in course_expansion_plan.json")

    if is_plan_managed_course and not course_title:
        errors.append("missing course_title field")

    if course_plan:
        expected_course_title = str(course_plan.get("course_title", "")).strip()
        if expected_course_title and course_title != expected_course_title:
            actual = course_title or "<empty>"
            errors.append(
                f"course_title mismatch for {source}: expected {expected_course_title}, got {actual}"
            )

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

        if entity_type not in allowed_entity_types:
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

    connected_local_nodes = {name for relation in relation_tuples for name in relation[:2]}
    isolated_nodes = sorted(name for name in current_name_set if name not in connected_local_nodes)
    if is_plan_managed_course and isolated_nodes:
        errors.append("nodes without relations: " + ", ".join(isolated_nodes))

    external_registry = load_external_node_registry(seed_path)
    external_name_to_source = {
        canonical_name: str(item.get("path", "")).strip()
        for canonical_name, item in external_registry.items()
        if str(item.get("path", "")).strip()
    }
    external_relations: set[tuple[str, str, str]] = set()
    for path in iter_registry_paths(seed_path):
        registry_payload = load_json_file(path)
        for relation in registry_payload.get("relations", []):
            if len(relation) == 3:
                external_relations.add(tuple(relation))

    reusable_reference_nodes = [
        f"{name} @ {external_name_to_source[name]}"
        for name in sorted(current_name_set)
        if name in external_name_to_source
    ]
    if reusable_reference_nodes:
        warnings.append(
            "canonical_name already defined in other curated files and will be treated as reusable references: "
            + "; ".join(reusable_reference_nodes)
        )

    metadata_conflicts: list[str] = []
    for node in nodes:
        canonical_name = str(node.get("canonical_name", "")).strip()
        if canonical_name not in external_registry:
            continue
        external = external_registry[canonical_name]
        current_entity_type = str(node.get("entity_type", "")).strip()
        current_description = str(node.get("description", "")).strip()
        external_entity_type = str(external.get("entity_type", "")).strip()
        external_description = str(external.get("description", "")).strip()
        if current_entity_type and external_entity_type and current_entity_type != external_entity_type:
            metadata_conflicts.append(
                f"{canonical_name}: entity_type {current_entity_type} != {external_entity_type}"
            )
        if current_description and external_description and current_description != external_description:
            metadata_conflicts.append(
                f"{canonical_name}: description differs from {external.get('path', '')}"
            )
    if metadata_conflicts:
        warnings.append(
            "reused node metadata differs from earlier curated definitions: "
            + "; ".join(sorted(metadata_conflicts))
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

    target_node_range: tuple[int, int] | None = None
    target_range = course_plan.get("target_node_range", []) if course_plan else []
    if isinstance(target_range, list) and len(target_range) == 2:
        try:
            target_node_range = (int(target_range[0]), int(target_range[1]))
        except (TypeError, ValueError):
            target_node_range = None

    if target_node_range:
        lower, upper = target_node_range
        if not (lower <= len(nodes) <= upper):
            warnings.append(
                f"node_count {len(nodes)} falls outside planned range {lower}-{upper} for {source}"
            )

    relation_density = round(len(relations) / len(nodes), 4) if nodes else 0.0
    if is_plan_managed_course and nodes:
        min_density, max_density = RELATION_DENSITY_RANGE
        if not (min_density <= relation_density <= max_density):
            warnings.append(
                "relation density "
                f"{relation_density:.2f} is outside recommended range {min_density:.1f}-{max_density:.1f}"
            )
        if not reusable_reference_nodes:
            warnings.append(
                "no reusable cross-course reference nodes detected; consider adding cross-course links"
            )

    return {
        "valid": not errors,
        "source": source,
        "node_count": len(nodes),
        "relation_count": len(relations),
        "metrics": {
            "relation_density": relation_density,
            "isolated_node_count": len(isolated_nodes),
            "reusable_reference_count": len(reusable_reference_nodes),
            "target_node_range": list(target_node_range) if target_node_range else None,
        },
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
