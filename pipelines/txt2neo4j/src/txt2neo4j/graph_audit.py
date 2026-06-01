from __future__ import annotations

import argparse
import json
from collections import Counter, defaultdict, deque
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from .models import ENTITY_TYPE_VALUES
from .neo4j_import import SUPPORTED_RELATION_TYPES
from .validate_curated_batch import RELATION_DENSITY_RANGE

PIPELINE_ROOT = Path(__file__).resolve().parents[2]
DEFAULT_REPORT_DIR = PIPELINE_ROOT / "artifacts" / "graph_quality"
TERM_DICTIONARY_PATH = PIPELINE_ROOT / "dictionaries" / "term_dictionary.json"

VALID_ENTITY_TYPES = set(ENTITY_TYPE_VALUES)

ADVANCED_PREREQUISITE_MARKERS = (
    "高级",
    "进阶",
    "应用",
    "部署",
    "工程",
    "优化",
    "大模型",
    "深度学习",
    "Transformer",
    "BERT",
    "RAG",
    "Agent",
    "CNN",
    "LSTM",
    "GRU",
    "PPO",
    "DQN",
    "GAN",
    "VAE",
    "Diffusion",
    "LoRA",
    "MLOps",
    "量化",
    "压缩",
)

FOUNDATIONAL_TARGET_MARKERS = (
    "基础",
    "入门",
    "概念",
    "原理",
    "定义",
    "语法",
    "数学基础",
    "高等数学",
    "线性代数",
    "微积分",
    "概率",
    "统计",
    "数据结构",
    "Python",
)


def load_json_file(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def add_issue(
    bucket: list[dict[str, Any]],
    code: str,
    message: str,
    *,
    scope: str = "global",
    details: dict[str, Any] | None = None,
) -> None:
    item: dict[str, Any] = {"code": code, "scope": scope, "message": message}
    if details:
        item["details"] = details
    bucket.append(item)


def iter_source_paths(pipeline_root: Path) -> list[Path]:
    config_dir = pipeline_root / "config"
    seed_path = config_dir / "curated_seed_graph.json"
    batch_dir = config_dir / "curated_batches"
    paths = [seed_path] if seed_path.exists() else []
    paths.extend(sorted(path for path in batch_dir.glob("*.json") if path.is_file()))
    return paths


def relation_tuple(raw: Any) -> tuple[str, str, str] | None:
    if not isinstance(raw, (list, tuple)) or len(raw) != 3:
        return None
    source, target, relation_type = (str(part).strip() for part in raw)
    if not source or not target or not relation_type:
        return None
    return source, target, relation_type


def load_graph_sources(pipeline_root: Path) -> list[dict[str, Any]]:
    sources: list[dict[str, Any]] = []
    for path in iter_source_paths(pipeline_root):
        payload = load_json_file(path)
        sources.append(
            {
                "path": path,
                "file": path.name,
                "source": str(payload.get("source", path.stem)).strip(),
                "course_title": str(payload.get("course_title", "")).strip(),
                "nodes": payload.get("nodes", []),
                "relations": payload.get("relations", []),
                "is_course_file": path.name.startswith("course_"),
            }
        )
    return sources


def load_course_plan(pipeline_root: Path) -> dict[str, Any]:
    path = pipeline_root / "config" / "course_expansion_plan.json"
    if not path.exists():
        return {"courses": []}
    return load_json_file(path)


def load_reference_taxonomy(pipeline_root: Path) -> dict[str, Any]:
    path = pipeline_root / "config" / "reference_taxonomy.json"
    if not path.exists():
        return {"domains": []}
    return load_json_file(path)


def collect_taxonomy_terms(taxonomy: dict[str, Any]) -> tuple[set[str], list[dict[str, Any]]]:
    terms: set[str] = set()
    domains: list[dict[str, Any]] = []
    for domain in taxonomy.get("domains", []):
        name = str(domain.get("name", "")).strip()
        children = [str(item).strip() for item in domain.get("children", []) if str(item).strip()]
        if name:
            terms.add(name)
            terms.update(children)
            domains.append({"name": name, "children": children})
    return terms, domains


def build_registry(sources: list[dict[str, Any]], errors: list[dict[str, Any]]) -> dict[str, list[dict[str, Any]]]:
    registry: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for source in sources:
        seen_in_file: set[str] = set()
        for index, node in enumerate(source["nodes"]):
            canonical_name = str(node.get("canonical_name", "")).strip()
            entity_type = str(node.get("entity_type", "")).strip()
            description = str(node.get("description", "")).strip()

            if not canonical_name:
                add_issue(
                    errors,
                    "node.empty_canonical_name",
                    "Node has an empty canonical_name.",
                    scope=source["file"],
                    details={"index": index},
                )
                continue

            if canonical_name in seen_in_file:
                add_issue(
                    errors,
                    "node.duplicate_in_file",
                    f"Duplicate node in one file: {canonical_name}",
                    scope=source["file"],
                )
            seen_in_file.add(canonical_name)

            if entity_type not in VALID_ENTITY_TYPES:
                add_issue(
                    errors,
                    "node.invalid_entity_type",
                    f"Invalid entity_type for {canonical_name}: {entity_type}",
                    scope=source["file"],
                )

            if not description:
                add_issue(
                    errors,
                    "node.empty_description",
                    f"Node missing description: {canonical_name}",
                    scope=source["file"],
                )

            registry[canonical_name].append(
                {
                    "file": source["file"],
                    "source": source["source"],
                    "entity_type": entity_type,
                    "description": description,
                }
            )
    return registry


def find_definition_conflicts(registry: dict[str, list[dict[str, Any]]]) -> list[dict[str, Any]]:
    conflicts: list[dict[str, Any]] = []
    for canonical_name, definitions in registry.items():
        entity_types = sorted({item["entity_type"] for item in definitions if item["entity_type"]})
        descriptions = sorted({item["description"] for item in definitions if item["description"]})
        if len(entity_types) <= 1 and len(descriptions) <= 1:
            continue
        conflicts.append(
            {
                "canonical_name": canonical_name,
                "entity_types": entity_types,
                "description_count": len(descriptions),
                "definitions": definitions,
            }
        )
    return conflicts


def audit_alias_collisions(
    graph_names: set[str],
    errors: list[dict[str, Any]],
    term_dictionary_path: Path = TERM_DICTIONARY_PATH,
) -> list[dict[str, str]]:
    if not term_dictionary_path.exists():
        return []

    try:
        dictionary = load_json_file(term_dictionary_path)
    except json.JSONDecodeError as exc:
        add_issue(
            errors,
            "dictionary.invalid_json",
            "Term dictionary is not valid JSON.",
            scope=str(term_dictionary_path),
            details={"error": str(exc)},
        )
        return []

    collisions: list[dict[str, str]] = []
    entries = dictionary if isinstance(dictionary, list) else []
    for item in entries:
        canonical_name = str(item.get("canonical_name", "")).strip()
        if canonical_name not in graph_names:
            continue
        for alias in item.get("aliases", []):
            alias_name = str(alias).strip()
            if alias_name and alias_name in graph_names:
                collisions.append({"alias": alias_name, "canonical_name": canonical_name})

    for collision in collisions:
        add_issue(
            errors,
            "dictionary.alias_canonical_collision",
            "A term alias is also present as an independent canonical node.",
            scope=str(term_dictionary_path),
            details=collision,
        )
    return collisions


def text_has_any_marker(text: str, markers: tuple[str, ...]) -> bool:
    return any(marker in text for marker in markers)


def find_prerequisite_direction_risks(sources: list[dict[str, Any]]) -> list[dict[str, Any]]:
    risks: list[dict[str, Any]] = []
    for source in sources:
        for raw in source["relations"]:
            parsed = relation_tuple(raw)
            if not parsed:
                continue
            source_name, target_name, relation_type = parsed
            if relation_type != "PREREQUISITE_OF":
                continue
            if text_has_any_marker(source_name, ADVANCED_PREREQUISITE_MARKERS) and text_has_any_marker(
                target_name,
                FOUNDATIONAL_TARGET_MARKERS,
            ):
                risks.append(
                    {
                        "file": source["file"],
                        "source": source_name,
                        "target": target_name,
                        "relation_type": relation_type,
                        "reason": "advanced-looking source points to foundational-looking target",
                    }
                )
    return risks


def audit_relations(
    sources: list[dict[str, Any]],
    graph_names: set[str],
    errors: list[dict[str, Any]],
) -> dict[str, Any]:
    relation_type_counts: Counter[str] = Counter()
    relation_counts: Counter[tuple[str, str, str]] = Counter()
    adjacency: dict[str, set[str]] = defaultdict(set)
    file_reports: list[dict[str, Any]] = []

    for source in sources:
        local_relation_count = 0
        local_errors = 0
        for index, raw in enumerate(source["relations"]):
            parsed = relation_tuple(raw)
            if not parsed:
                local_errors += 1
                add_issue(
                    errors,
                    "relation.invalid_shape",
                    "Relation record must be a three-item [source, target, type] array with non-empty values.",
                    scope=source["file"],
                    details={"index": index, "record": raw},
                )
                continue

            source_name, target_name, relation_type = parsed
            local_relation_count += 1
            relation_type_counts[relation_type] += 1
            relation_counts[parsed] += 1

            if relation_type not in SUPPORTED_RELATION_TYPES:
                local_errors += 1
                add_issue(
                    errors,
                    "relation.unsupported_type",
                    f"Unsupported relation type: {relation_type}",
                    scope=source["file"],
                    details={"source": source_name, "target": target_name},
                )
            if source_name == target_name:
                local_errors += 1
                add_issue(
                    errors,
                    "relation.self_loop",
                    f"Self-loop relation: {source_name}",
                    scope=source["file"],
                    details={"relation_type": relation_type},
                )
            if source_name not in graph_names:
                local_errors += 1
                add_issue(
                    errors,
                    "relation.missing_source",
                    f"Relation source is not defined by any curated node: {source_name}",
                    scope=source["file"],
                    details={"target": target_name, "relation_type": relation_type},
                )
            if target_name not in graph_names:
                local_errors += 1
                add_issue(
                    errors,
                    "relation.missing_target",
                    f"Relation target is not defined by any curated node: {target_name}",
                    scope=source["file"],
                    details={"source": source_name, "relation_type": relation_type},
                )

            if source_name in graph_names and target_name in graph_names and source_name != target_name:
                adjacency[source_name].add(target_name)
                adjacency[target_name].add(source_name)

        node_count = len(source["nodes"])
        file_reports.append(
            {
                "file": source["file"],
                "source": source["source"],
                "course_title": source["course_title"],
                "node_count": node_count,
                "relation_count": local_relation_count,
                "relation_density": round(local_relation_count / node_count, 4) if node_count else 0.0,
                "relation_error_count": local_errors,
            }
        )

    duplicate_relation_groups = [
        {
            "source": source,
            "target": target,
            "relation_type": relation_type,
            "count": count,
        }
        for (source, target, relation_type), count in relation_counts.items()
        if count > 1
    ]

    return {
        "relation_type_counts": dict(sorted(relation_type_counts.items())),
        "raw_relation_count": sum(relation_type_counts.values()),
        "unique_relation_count": len(relation_counts),
        "duplicate_relation_group_count": len(duplicate_relation_groups),
        "duplicate_relation_samples": sorted(
            duplicate_relation_groups,
            key=lambda item: (-item["count"], item["source"], item["target"], item["relation_type"]),
        )[:30],
        "adjacency": adjacency,
        "file_reports": file_reports,
    }


def compute_connectivity(graph_names: set[str], adjacency: dict[str, set[str]]) -> dict[str, Any]:
    unseen = set(graph_names)
    component_sizes: list[int] = []
    while unseen:
        start = unseen.pop()
        queue: deque[str] = deque([start])
        size = 0
        while queue:
            current = queue.popleft()
            size += 1
            for neighbor in adjacency.get(current, set()):
                if neighbor in unseen:
                    unseen.remove(neighbor)
                    queue.append(neighbor)
        component_sizes.append(size)

    component_sizes.sort(reverse=True)
    isolated_nodes = sorted(name for name in graph_names if not adjacency.get(name))
    largest = component_sizes[0] if component_sizes else 0
    return {
        "component_count": len(component_sizes),
        "largest_component_size": largest,
        "largest_component_ratio": round(largest / len(graph_names), 4) if graph_names else 1.0,
        "component_sizes": component_sizes[:20],
        "isolated_node_count": len(isolated_nodes),
        "isolated_node_samples": isolated_nodes[:50],
    }


def audit_course_plan(
    plan: dict[str, Any],
    sources: list[dict[str, Any]],
    errors: list[dict[str, Any]],
    warnings: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    source_lookup: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for source in sources:
        if source["source"]:
            source_lookup[source["source"]].append(source)

    course_reports: list[dict[str, Any]] = []
    for course in plan.get("courses", []):
        source = str(course.get("source", "")).strip()
        expected_title = str(course.get("course_title", "")).strip()
        target_range = course.get("target_node_range", [])
        matches = source_lookup.get(source, [])
        if not matches:
            add_issue(
                errors,
                "course.missing_curated_file",
                f"Course plan source is missing from curated batches: {source}",
                scope=source or "course_plan",
            )
            course_reports.append(
                {
                    "source": source,
                    "course_title": expected_title,
                    "status": "missing",
                    "node_count": 0,
                    "relation_count": 0,
                    "relation_density": 0.0,
                    "target_node_range": target_range,
                }
            )
            continue

        if len(matches) > 1:
            add_issue(
                errors,
                "course.duplicate_source",
                f"Course source appears in multiple files: {source}",
                scope=source,
                details={"files": [item["file"] for item in matches]},
            )

        item = matches[0]
        node_count = len(item["nodes"])
        relation_count = len([raw for raw in item["relations"] if relation_tuple(raw)])
        relation_density = round(relation_count / node_count, 4) if node_count else 0.0
        status = "pass"

        if expected_title and item["course_title"] != expected_title:
            status = "fail"
            add_issue(
                errors,
                "course.title_mismatch",
                f"Course title mismatch for {source}.",
                scope=item["file"],
                details={"expected": expected_title, "actual": item["course_title"]},
            )

        if isinstance(target_range, list) and len(target_range) == 2:
            lower, upper = int(target_range[0]), int(target_range[1])
            if not lower <= node_count <= upper:
                status = "fail"
                add_issue(
                    errors,
                    "course.node_count_out_of_range",
                    f"Node count for {source} falls outside the planned range.",
                    scope=item["file"],
                    details={"node_count": node_count, "target_node_range": target_range},
                )

        min_density, max_density = RELATION_DENSITY_RANGE
        if node_count and not min_density <= relation_density <= max_density:
            add_issue(
                warnings,
                "course.relation_density_out_of_range",
                f"Relation density for {source} is outside the recommended range.",
                scope=item["file"],
                details={
                    "relation_density": relation_density,
                    "recommended_range": [min_density, max_density],
                },
            )

        course_reports.append(
            {
                "source": source,
                "course_title": expected_title,
                "file": item["file"],
                "status": status,
                "node_count": node_count,
                "relation_count": relation_count,
                "relation_density": relation_density,
                "target_node_range": target_range,
            }
        )

    return course_reports


def build_graph_quality_report(pipeline_root: Path = PIPELINE_ROOT) -> dict[str, Any]:
    pipeline_root = pipeline_root.resolve()
    errors: list[dict[str, Any]] = []
    warnings: list[dict[str, Any]] = []

    sources = load_graph_sources(pipeline_root)
    plan = load_course_plan(pipeline_root)
    taxonomy = load_reference_taxonomy(pipeline_root)

    registry = build_registry(sources, errors)
    graph_names = set(registry)

    conflicts = find_definition_conflicts(registry)
    for conflict in conflicts:
        add_issue(
            errors,
            "node.definition_conflict",
            f"Conflicting definitions for {conflict['canonical_name']}",
            scope="registry",
            details=conflict,
        )
    alias_collisions = audit_alias_collisions(
        graph_names,
        errors,
        pipeline_root / "dictionaries" / "term_dictionary.json",
    )

    relation_report = audit_relations(sources, graph_names, errors)
    connectivity = compute_connectivity(graph_names, relation_report["adjacency"])
    if connectivity["isolated_node_count"] > 0:
        add_issue(
            errors,
            "graph.isolated_nodes",
            "Curated graph contains isolated nodes.",
            details={"samples": connectivity["isolated_node_samples"]},
        )
    if connectivity["component_count"] > 1:
        add_issue(
            warnings,
            "graph.multiple_components",
            "Curated graph is split into multiple weakly connected components.",
            details={"component_sizes": connectivity["component_sizes"]},
        )

    course_reports = audit_course_plan(plan, sources, errors, warnings)

    taxonomy_terms, taxonomy_domains = collect_taxonomy_terms(taxonomy)
    missing_taxonomy_terms = sorted(taxonomy_terms - graph_names)
    if missing_taxonomy_terms:
        add_issue(
            errors,
            "taxonomy.missing_terms",
            "Reference taxonomy contains terms missing from curated graph nodes.",
            scope="reference_taxonomy",
            details={"missing_terms": missing_taxonomy_terms},
        )

    domain_reports: list[dict[str, Any]] = []
    for domain in taxonomy_domains:
        terms = [domain["name"], *domain["children"]]
        covered = [term for term in terms if term in graph_names]
        missing = [term for term in terms if term not in graph_names]
        domain_reports.append(
            {
                "name": domain["name"],
                "term_count": len(terms),
                "covered_count": len(covered),
                "coverage_ratio": round(len(covered) / len(terms), 4) if terms else 1.0,
                "missing_terms": missing,
            }
        )

    prerequisite_risks = find_prerequisite_direction_risks(sources)
    for risk in prerequisite_risks:
        add_issue(
            warnings,
            "relation.prerequisite_direction_review",
            "Potential reversed prerequisite relation needs manual review.",
            scope=risk["file"],
            details=risk,
        )

    reused_nodes = [
        {"canonical_name": name, "definition_count": len(definitions)}
        for name, definitions in registry.items()
        if len(definitions) > 1
    ]
    reused_nodes.sort(key=lambda item: (-item["definition_count"], item["canonical_name"]))

    course_count = len(plan.get("courses", []))
    course_pass_count = sum(1 for item in course_reports if item["status"] == "pass")

    summary = {
        "file_count": len(sources),
        "course_plan_count": course_count,
        "course_pass_count": course_pass_count,
        "node_count": len(graph_names),
        "raw_relation_count": relation_report["raw_relation_count"],
        "unique_relation_count": relation_report["unique_relation_count"],
        "taxonomy_term_count": len(taxonomy_terms),
        "taxonomy_covered_count": len(taxonomy_terms) - len(missing_taxonomy_terms),
        "taxonomy_coverage_ratio": round(
            (len(taxonomy_terms) - len(missing_taxonomy_terms)) / len(taxonomy_terms),
            4,
        )
        if taxonomy_terms
        else 1.0,
        "component_count": connectivity["component_count"],
        "largest_component_ratio": connectivity["largest_component_ratio"],
        "definition_conflict_count": len(conflicts),
        "alias_collision_count": len(alias_collisions),
        "prerequisite_direction_risk_count": len(prerequisite_risks),
        "error_count": len(errors),
        "warning_count": len(warnings),
    }

    return {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "pipeline_root": str(pipeline_root),
        "gate_status": "pass" if not errors else "fail",
        "summary": summary,
        "quality_gates": {
            "course_plan": {
                "course_count": course_count,
                "pass_count": course_pass_count,
                "coverage_ratio": round(course_pass_count / course_count, 4) if course_count else 1.0,
                "recommended_relation_density_range": list(RELATION_DENSITY_RANGE),
            },
            "taxonomy": {
                "term_count": len(taxonomy_terms),
                "covered_count": summary["taxonomy_covered_count"],
                "coverage_ratio": summary["taxonomy_coverage_ratio"],
                "missing_terms": missing_taxonomy_terms,
                "domains": domain_reports,
            },
            "definitions": {
                "reused_node_count": len(reused_nodes),
                "conflict_count": len(conflicts),
                "top_reused_nodes": reused_nodes[:30],
            },
            "dictionary": {
                "alias_collision_count": len(alias_collisions),
                "alias_collisions": alias_collisions,
            },
            "relations": {
                "relation_type_counts": relation_report["relation_type_counts"],
                "raw_relation_count": relation_report["raw_relation_count"],
                "unique_relation_count": relation_report["unique_relation_count"],
                "duplicate_relation_group_count": relation_report["duplicate_relation_group_count"],
                "duplicate_relation_samples": relation_report["duplicate_relation_samples"],
                "prerequisite_direction_risks": prerequisite_risks,
            },
            "connectivity": connectivity,
        },
        "courses": course_reports,
        "files": relation_report["file_reports"],
        "errors": errors,
        "warnings": warnings,
    }


def markdown_table(rows: list[list[str]]) -> list[str]:
    if not rows:
        return []
    widths = [max(len(str(row[index])) for row in rows) for index in range(len(rows[0]))]
    result: list[str] = []
    for row_index, row in enumerate(rows):
        result.append("| " + " | ".join(str(value).ljust(widths[index]) for index, value in enumerate(row)) + " |")
        if row_index == 0:
            result.append("| " + " | ".join("-" * widths[index] for index in range(len(row))) + " |")
    return result


def render_markdown_report(report: dict[str, Any]) -> str:
    summary = report["summary"]
    gates = report["quality_gates"]
    lines = [
        "# 知识图谱质量审计报告",
        "",
        f"- 生成时间：`{report['generated_at']}`",
        f"- 门禁状态：`{report['gate_status'].upper()}`",
        f"- 源文件数：`{summary['file_count']}`",
        f"- 课程覆盖：`{summary['course_pass_count']}/{summary['course_plan_count']}`",
        f"- 节点数：`{summary['node_count']}`",
        f"- 关系数：`{summary['raw_relation_count']}`，去重后 `{summary['unique_relation_count']}`",
        f"- Taxonomy 覆盖率：`{summary['taxonomy_coverage_ratio']:.2%}`",
        f"- 连通分量：`{summary['component_count']}`，最大分量占比 `{summary['largest_component_ratio']:.2%}`",
        f"- 定义冲突：`{summary['definition_conflict_count']}`",
        f"- 别名 canonical 冲突：`{summary['alias_collision_count']}`",
        f"- 先修方向风险：`{summary['prerequisite_direction_risk_count']}`",
        "",
        "## 课程覆盖",
        "",
    ]

    course_rows = [["课程", "状态", "节点", "关系", "密度", "目标范围"]]
    for course in report["courses"]:
        target_range = course.get("target_node_range") or []
        course_rows.append(
            [
                course.get("course_title") or course.get("source", ""),
                course.get("status", ""),
                str(course.get("node_count", 0)),
                str(course.get("relation_count", 0)),
                f"{float(course.get('relation_density', 0.0)):.2f}",
                "-".join(str(item) for item in target_range) if target_range else "-",
            ]
        )
    lines.extend(markdown_table(course_rows))

    lines.extend(
        [
            "",
            "## Taxonomy 覆盖",
            "",
            f"- 覆盖 `{gates['taxonomy']['covered_count']}/{gates['taxonomy']['term_count']}` 个参考主题。",
        ]
    )
    if gates["taxonomy"]["missing_terms"]:
        lines.append("- 缺失主题：" + "、".join(gates["taxonomy"]["missing_terms"]))
    else:
        lines.append("- 缺失主题：无")

    lines.extend(["", "## 关系类型分布", ""])
    relation_rows = [["关系类型", "数量"]]
    for relation_type, count in gates["relations"]["relation_type_counts"].items():
        relation_rows.append([relation_type, str(count)])
    lines.extend(markdown_table(relation_rows))

    lines.extend(
        [
            "",
            "## 一致性结论",
            "",
            f"- 跨文件复用节点：`{gates['definitions']['reused_node_count']}` 个。",
            f"- 同名定义冲突：`{gates['definitions']['conflict_count']}` 个。",
            f"- 重复关系组：`{gates['relations']['duplicate_relation_group_count']}` 个；导入侧使用 MERGE，重复边不会造成多重写入。",
            f"- 孤立节点：`{gates['connectivity']['isolated_node_count']}` 个。",
        ]
    )

    if report["errors"]:
        lines.extend(["", "## Errors", ""])
        for item in report["errors"][:50]:
            lines.append(f"- `{item['code']}` [{item['scope']}]: {item['message']}")
    if report["warnings"]:
        lines.extend(["", "## Warnings", ""])
        for item in report["warnings"][:50]:
            lines.append(f"- `{item['code']}` [{item['scope']}]: {item['message']}")

    return "\n".join(lines).rstrip() + "\n"


def write_report(report: dict[str, Any], output_dir: Path) -> dict[str, str]:
    output_dir.mkdir(parents=True, exist_ok=True)
    json_path = output_dir / "knowledge_graph_audit.json"
    markdown_path = output_dir / "knowledge_graph_audit.md"
    json_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    markdown_path.write_text(render_markdown_report(report), encoding="utf-8")
    return {"json_path": str(json_path), "markdown_path": str(markdown_path)}


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Audit curated course knowledge graph assets.")
    parser.add_argument("--pipeline-root", default=str(PIPELINE_ROOT))
    parser.add_argument("--output-dir", default=str(DEFAULT_REPORT_DIR))
    parser.add_argument("--fail-on-warning", action="store_true")
    return parser


def main() -> None:
    args = build_parser().parse_args()
    report = build_graph_quality_report(Path(args.pipeline_root))
    paths = write_report(report, Path(args.output_dir))
    payload = {
        "gate_status": report["gate_status"],
        "summary": report["summary"],
        **paths,
    }
    print(json.dumps(payload, ensure_ascii=False, indent=2))
    if report["errors"] or (args.fail_on_warning and report["warnings"]):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
