from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[1]
SRC_ROOT = PROJECT_ROOT / "src"

if str(SRC_ROOT) not in sys.path:
    sys.path.insert(0, str(SRC_ROOT))

from txt2neo4j.graph_audit import build_graph_quality_report, render_markdown_report  # noqa: E402


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def write_seed(root: Path) -> None:
    write_json(root / "config" / "curated_seed_graph.json", {"source": "seed", "nodes": [], "relations": []})


class GraphAuditTest(unittest.TestCase):
    def test_report_passes_complete_connected_course_graph(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            write_seed(root)
            write_json(
                root / "config" / "course_expansion_plan.json",
                {
                    "courses": [
                        {
                            "source": "course_machine_learning",
                            "course_title": "机器学习",
                            "target_node_range": [4, 4],
                        }
                    ]
                },
            )
            write_json(
                root / "config" / "reference_taxonomy.json",
                {
                    "domains": [
                        {
                            "name": "机器学习",
                            "children": ["监督学习", "线性回归", "模型评估"],
                        }
                    ]
                },
            )
            write_json(
                root / "config" / "curated_batches" / "course_machine_learning.json",
                {
                    "source": "course_machine_learning",
                    "course_title": "机器学习",
                    "nodes": [
                        {
                            "canonical_name": "机器学习",
                            "entity_type": "topic",
                            "description": "通过数据学习模型和规律的人工智能核心方向。",
                        },
                        {
                            "canonical_name": "监督学习",
                            "entity_type": "method",
                            "description": "利用带标签样本学习输入到输出映射的学习范式。",
                        },
                        {
                            "canonical_name": "线性回归",
                            "entity_type": "algorithm",
                            "description": "用线性函数拟合连续数值输出的监督学习算法。",
                        },
                        {
                            "canonical_name": "模型评估",
                            "entity_type": "method",
                            "description": "使用指标和验证策略衡量模型泛化能力的过程。",
                        },
                    ],
                    "relations": [
                        ["监督学习", "机器学习", "SUBTOPIC_OF"],
                        ["线性回归", "监督学习", "SUBTOPIC_OF"],
                        ["模型评估", "机器学习", "SUBTOPIC_OF"],
                        ["监督学习", "线性回归", "PREREQUISITE_OF"],
                        ["模型评估", "线性回归", "RELATED_TO"],
                        ["线性回归", "机器学习", "APPLIES_TO"],
                    ],
                },
            )

            report = build_graph_quality_report(root)

            self.assertEqual(report["gate_status"], "pass")
            self.assertEqual(report["summary"]["node_count"], 4)
            self.assertEqual(report["summary"]["taxonomy_coverage_ratio"], 1.0)
            self.assertEqual(report["summary"]["component_count"], 1)
            self.assertEqual(report["summary"]["error_count"], 0)
            self.assertEqual(report["summary"]["warning_count"], 0)
            self.assertIn("知识图谱质量审计报告", render_markdown_report(report))

    def test_report_detects_conflicting_definitions_and_missing_relation_endpoint(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            write_seed(root)
            write_json(
                root / "config" / "course_expansion_plan.json",
                {
                    "courses": [
                        {
                            "source": "course_machine_learning",
                            "course_title": "机器学习",
                            "target_node_range": [2, 2],
                        }
                    ]
                },
            )
            write_json(
                root / "config" / "reference_taxonomy.json",
                {"domains": [{"name": "机器学习", "children": ["共享节点"]}]},
            )
            write_json(
                root / "config" / "curated_batches" / "course_machine_learning.json",
                {
                    "source": "course_machine_learning",
                    "course_title": "机器学习",
                    "nodes": [
                        {
                            "canonical_name": "机器学习",
                            "entity_type": "topic",
                            "description": "通过数据学习模型和规律的人工智能核心方向。",
                        },
                        {
                            "canonical_name": "共享节点",
                            "entity_type": "concept",
                            "description": "第一个文件中的概念定义。",
                        },
                    ],
                    "relations": [["共享节点", "不存在节点", "RELATED_TO"]],
                },
            )
            write_json(
                root / "config" / "curated_batches" / "batch_reference.json",
                {
                    "source": "batch_reference",
                    "nodes": [
                        {
                            "canonical_name": "共享节点",
                            "entity_type": "method",
                            "description": "第二个文件中的不同定义。",
                        }
                    ],
                    "relations": [],
                },
            )

            report = build_graph_quality_report(root)
            error_codes = {item["code"] for item in report["errors"]}

            self.assertEqual(report["gate_status"], "fail")
            self.assertIn("node.definition_conflict", error_codes)
            self.assertIn("relation.missing_target", error_codes)

    def test_report_flags_suspicious_prerequisite_direction_as_warning(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            write_seed(root)
            write_json(
                root / "config" / "course_expansion_plan.json",
                {
                    "courses": [
                        {
                            "source": "course_llm",
                            "course_title": "大模型",
                            "target_node_range": [2, 2],
                        }
                    ]
                },
            )
            write_json(
                root / "config" / "reference_taxonomy.json",
                {"domains": [{"name": "大模型", "children": ["基础概念"]}]},
            )
            write_json(
                root / "config" / "curated_batches" / "course_llm.json",
                {
                    "source": "course_llm",
                    "course_title": "大模型",
                    "nodes": [
                        {
                            "canonical_name": "大模型",
                            "entity_type": "topic",
                            "description": "以大规模参数和语料预训练获得通用能力的模型体系。",
                        },
                        {
                            "canonical_name": "基础概念",
                            "entity_type": "concept",
                            "description": "学习课程前需要掌握的基本概念集合。",
                        },
                    ],
                    "relations": [
                        ["基础概念", "大模型", "SUBTOPIC_OF"],
                        ["基础概念", "大模型", "RELATED_TO"],
                        ["大模型", "基础概念", "PREREQUISITE_OF"],
                    ],
                },
            )

            report = build_graph_quality_report(root)
            warning_codes = {item["code"] for item in report["warnings"]}

            self.assertEqual(report["gate_status"], "pass")
            self.assertIn("relation.prerequisite_direction_review", warning_codes)

    def test_report_rejects_alias_used_as_independent_canonical_node(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            write_json(
                root / "config" / "curated_seed_graph.json",
                {
                    "source": "seed",
                    "nodes": [
                        {
                            "canonical_name": "A*搜索算法",
                            "entity_type": "algorithm",
                            "description": "使用启发式函数和路径代价进行最优优先扩展的搜索算法。",
                        },
                        {
                            "canonical_name": "A*搜索",
                            "entity_type": "algorithm",
                            "description": "A*搜索算法的别名节点，不应作为独立 canonical 节点出现。",
                        },
                    ],
                    "relations": [["A*搜索", "A*搜索算法", "RELATED_TO"]],
                },
            )
            write_json(
                root / "dictionaries" / "term_dictionary.json",
                [
                    {
                        "canonical_name": "A*搜索算法",
                        "entity_type": "algorithm",
                        "aliases": ["A*搜索", "A星搜索算法"],
                    }
                ],
            )

            report = build_graph_quality_report(root)
            error_codes = {item["code"] for item in report["errors"]}

            self.assertEqual(report["gate_status"], "fail")
            self.assertIn("dictionary.alias_canonical_collision", error_codes)
            self.assertEqual(report["summary"]["alias_collision_count"], 1)

    def test_report_accepts_modern_ai_entity_types(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            write_seed(
                root,
            )
            write_json(
                root / "config" / "course_expansion_plan.json",
                {"courses": []},
            )
            write_json(
                root / "config" / "reference_taxonomy.json",
                {"domains": []},
            )
            write_json(
                root / "config" / "curated_batches" / "frontier_modern_types.json",
                {
                    "source": "frontier_modern_types",
                    "nodes": [
                        {
                            "canonical_name": "GPT-4",
                            "entity_type": "model",
                            "description": "具备多任务语言理解和生成能力的大规模模型。",
                        },
                        {
                            "canonical_name": "vLLM",
                            "entity_type": "tool",
                            "description": "用于高吞吐大模型推理服务的工程工具。",
                        },
                        {
                            "canonical_name": "LangChain",
                            "entity_type": "framework",
                            "description": "用于编排大模型应用和工具调用的开发框架。",
                        },
                    ],
                    "relations": [
                        ["vLLM", "GPT-4", "RELATED_TO"],
                        ["LangChain", "GPT-4", "USES"],
                    ],
                },
            )

            report = build_graph_quality_report(root)

            self.assertEqual(report["gate_status"], "pass")
            self.assertEqual(report["summary"]["error_count"], 0)


if __name__ == "__main__":
    unittest.main()
