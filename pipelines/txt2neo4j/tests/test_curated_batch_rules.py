from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

PROJECT_ROOT = Path(__file__).resolve().parents[1]
SRC_ROOT = PROJECT_ROOT / "src"

if str(SRC_ROOT) not in sys.path:
    sys.path.insert(0, str(SRC_ROOT))

from txt2neo4j.curated_graph import build_curated_rows, resolve_seed_paths_from_pattern  # noqa: E402
from txt2neo4j.normalize_graph import load_curated_registry  # noqa: E402
from txt2neo4j.validate_curated_batch import validate_seed_path  # noqa: E402


def write_json(path: Path, payload: dict) -> None:
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")


def write_course_plan(path: Path, courses: list[dict]) -> None:
    write_json(
        path,
        {
            "plan_name": "test_course_expansion_plan",
            "courses": courses,
        },
    )


class CuratedBatchRulesTest(unittest.TestCase):
    def test_resolve_seed_paths_from_pattern_supports_exact_name_and_glob(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            batch_dir = root / "curated_batches"
            batch_dir.mkdir()

            write_json(
                batch_dir / "course_llm.json",
                {"source": "course_llm", "course_title": "大模型", "nodes": [], "relations": []},
            )
            write_json(
                batch_dir / "course_ai_ethics.json",
                {"source": "course_ai_ethics", "course_title": "AI伦理", "nodes": [], "relations": []},
            )

            with patch("txt2neo4j.curated_graph.CURATED_BATCH_DIR", batch_dir):
                exact_paths = resolve_seed_paths_from_pattern("course_llm")
                glob_paths = resolve_seed_paths_from_pattern("course_*.json")

            self.assertEqual([path.name for path in exact_paths], ["course_llm.json"])
            self.assertEqual(
                [path.name for path in glob_paths],
                ["course_ai_ethics.json", "course_llm.json"],
            )

    def test_validate_seed_allows_reusable_reference_nodes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            seed_path = root / "curated_seed_graph.json"
            batch_dir = root / "curated_batches"
            plan_path = root / "course_expansion_plan.json"
            batch_dir.mkdir()

            write_json(seed_path, {"source": "seed_root", "nodes": [], "relations": []})
            write_course_plan(
                plan_path,
                [
                    {
                        "source": "course_calculus",
                        "course_title": "高等数学",
                        "target_node_range": [100, 120],
                    }
                ],
            )
            write_json(
                batch_dir / "batch_28_machine_learning_full_route.json",
                {
                    "source": "course_machine_learning",
                    "course_title": "机器学习",
                    "nodes": [
                        {
                            "canonical_name": "梯度下降",
                            "entity_type": "method",
                            "description": "通过迭代更新参数逼近目标函数最优解的优化方法。",
                        }
                    ],
                    "relations": [],
                },
            )

            target_path = batch_dir / "course_calculus.json"
            write_json(
                target_path,
                {
                    "source": "course_calculus",
                    "course_title": "高等数学",
                    "nodes": [
                        {
                            "canonical_name": "梯度下降",
                            "entity_type": "method",
                            "description": "复用机器学习中的优化节点，用来建立跨课程先修关系。",
                        },
                        {
                            "canonical_name": "链式法则",
                            "entity_type": "method",
                            "description": "复合函数求导时逐层传递导数的求导法则。",
                        },
                    ],
                    "relations": [["链式法则", "梯度下降", "PREREQUISITE_OF"]],
                },
            )

            with patch("txt2neo4j.validate_curated_batch.CURATED_SEED_PATH", seed_path), patch(
                "txt2neo4j.validate_curated_batch.CURATED_BATCH_DIR", batch_dir
            ), patch(
                "txt2neo4j.validate_curated_batch.COURSE_EXPANSION_PLAN_PATH", plan_path
            ):
                result = validate_seed_path(target_path)

            self.assertTrue(result["valid"])
            self.assertEqual(result["errors"], [])
            self.assertTrue(
                any("reusable references" in warning for warning in result["warnings"])
            )
            self.assertEqual(result["metrics"]["reusable_reference_count"], 1)

    def test_build_curated_rows_prefers_external_definition_for_reused_node(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            seed_path = root / "curated_seed_graph.json"
            batch_dir = root / "curated_batches"
            batch_dir.mkdir()

            write_json(seed_path, {"source": "seed_root", "nodes": [], "relations": []})
            write_json(
                batch_dir / "batch_28_machine_learning_full_route.json",
                {
                    "source": "course_machine_learning",
                    "course_title": "机器学习",
                    "nodes": [
                        {
                            "canonical_name": "梯度下降",
                            "entity_type": "method",
                            "description": "通过迭代更新参数逼近目标函数最优解的优化方法。",
                        }
                    ],
                    "relations": [],
                },
            )

            target_path = batch_dir / "course_calculus.json"
            write_json(
                target_path,
                {
                    "source": "course_calculus",
                    "course_title": "高等数学",
                    "nodes": [
                        {
                            "canonical_name": "梯度下降",
                            "entity_type": "algorithm",
                            "description": "当前课程中的引用节点，不应该覆盖机器学习里的原始定义。",
                        }
                    ],
                    "relations": [],
                },
            )

            with patch("txt2neo4j.curated_graph.CURATED_SEED_PATH", seed_path), patch(
                "txt2neo4j.curated_graph.CURATED_BATCH_DIR", batch_dir
            ):
                node_rows, relation_rows, seed = build_curated_rows({}, target_path)

            self.assertEqual(seed["source"], "course_calculus")
            self.assertEqual(relation_rows, [])
            self.assertEqual(len(node_rows), 1)
            self.assertEqual(node_rows[0]["entity_type"], "method")
            self.assertEqual(node_rows[0]["description"], "通过迭代更新参数逼近目标函数最优解的优化方法。")
            self.assertEqual(node_rows[0]["chapter_title"], "机器学习")
            self.assertEqual(node_rows[0]["source_path"], "curated/course_machine_learning")
            self.assertEqual(node_rows[0]["curated_source"], "course_machine_learning")

    def test_load_curated_registry_keeps_first_definition_for_duplicate_nodes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            seed_path = root / "curated_seed_graph.json"
            batch_dir = root / "curated_batches"
            batch_dir.mkdir()

            write_json(seed_path, {"source": "seed_root", "nodes": [], "relations": []})
            write_json(
                batch_dir / "batch_28_machine_learning_full_route.json",
                {
                    "source": "course_machine_learning",
                    "course_title": "机器学习",
                    "nodes": [
                        {
                            "canonical_name": "梯度下降",
                            "entity_type": "method",
                            "description": "通过迭代更新参数逼近目标函数最优解的优化方法。",
                        }
                    ],
                    "relations": [],
                },
            )
            write_json(
                batch_dir / "course_calculus.json",
                {
                    "source": "course_calculus",
                    "course_title": "高等数学",
                    "nodes": [
                        {
                            "canonical_name": "梯度下降",
                            "entity_type": "algorithm",
                            "description": "后写课程中的引用定义，不应反向覆盖已有课程定义。",
                        }
                    ],
                    "relations": [],
                },
            )

            with patch("txt2neo4j.normalize_graph.CURATED_SEED_PATH", seed_path), patch(
                "txt2neo4j.normalize_graph.CURATED_BATCH_DIR", batch_dir
            ):
                registry, relations = load_curated_registry()

            self.assertEqual(relations, set())
            self.assertEqual(registry["梯度下降"]["entity_type"], "method")
            self.assertEqual(
                registry["梯度下降"]["description"],
                "通过迭代更新参数逼近目标函数最优解的优化方法。",
            )
            self.assertEqual(registry["梯度下降"]["course_title"], "机器学习")
            self.assertEqual(registry["梯度下降"]["source"], "course_machine_learning")

    def test_validate_seed_checks_plan_metadata_and_quality_warnings(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            seed_path = root / "curated_seed_graph.json"
            batch_dir = root / "curated_batches"
            plan_path = root / "course_expansion_plan.json"
            batch_dir.mkdir()

            write_json(seed_path, {"source": "seed_root", "nodes": [], "relations": []})
            write_course_plan(
                plan_path,
                [
                    {
                        "source": "course_linear_algebra",
                        "course_title": "线性代数",
                        "target_node_range": [80, 100],
                    }
                ],
            )

            target_path = batch_dir / "course_linear_algebra.json"
            write_json(
                target_path,
                {
                    "source": "course_linear_algebra",
                    "course_title": "线性代数",
                    "nodes": [
                        {
                            "canonical_name": "线性代数",
                            "entity_type": "topic",
                            "description": "研究向量空间、矩阵与线性映射的基础数学课程。",
                        },
                        {
                            "canonical_name": "矩阵",
                            "entity_type": "concept",
                            "description": "按行列排列并用于表示线性变换的数据结构。",
                        },
                    ],
                    "relations": [["矩阵", "线性代数", "SUBTOPIC_OF"]],
                },
            )

            with patch("txt2neo4j.validate_curated_batch.CURATED_SEED_PATH", seed_path), patch(
                "txt2neo4j.validate_curated_batch.CURATED_BATCH_DIR", batch_dir
            ), patch(
                "txt2neo4j.validate_curated_batch.COURSE_EXPANSION_PLAN_PATH", plan_path
            ):
                result = validate_seed_path(target_path)

            self.assertTrue(result["valid"])
            self.assertEqual(result["errors"], [])
            self.assertAlmostEqual(result["metrics"]["relation_density"], 0.5)
            self.assertEqual(result["metrics"]["target_node_range"], [80, 100])
            self.assertTrue(
                any("planned range 80-100" in warning for warning in result["warnings"])
            )
            self.assertTrue(
                any("relation density 0.50" in warning for warning in result["warnings"])
            )
            self.assertTrue(
                any("no reusable cross-course reference nodes detected" in warning for warning in result["warnings"])
            )

    def test_validate_seed_rejects_wrong_course_title_and_orphan_nodes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            seed_path = root / "curated_seed_graph.json"
            batch_dir = root / "curated_batches"
            plan_path = root / "course_expansion_plan.json"
            batch_dir.mkdir()

            write_json(seed_path, {"source": "seed_root", "nodes": [], "relations": []})
            write_course_plan(
                plan_path,
                [
                    {
                        "source": "course_calculus",
                        "course_title": "高等数学",
                        "target_node_range": [100, 120],
                    }
                ],
            )

            target_path = batch_dir / "course_calculus.json"
            write_json(
                target_path,
                {
                    "source": "course_calculus",
                    "course_title": "数学分析",
                    "nodes": [
                        {
                            "canonical_name": "高等数学",
                            "entity_type": "topic",
                            "description": "研究极限、微分和积分等连续变化规律的课程。",
                        },
                        {
                            "canonical_name": "极限",
                            "entity_type": "concept",
                            "description": "刻画变量逼近某一值时趋势的基础分析概念。",
                        },
                        {
                            "canonical_name": "孤立节点",
                            "entity_type": "concept",
                            "description": "故意不连任何关系以触发质量校验的测试节点。",
                        },
                    ],
                    "relations": [["极限", "高等数学", "SUBTOPIC_OF"]],
                },
            )

            with patch("txt2neo4j.validate_curated_batch.CURATED_SEED_PATH", seed_path), patch(
                "txt2neo4j.validate_curated_batch.CURATED_BATCH_DIR", batch_dir
            ), patch(
                "txt2neo4j.validate_curated_batch.COURSE_EXPANSION_PLAN_PATH", plan_path
            ):
                result = validate_seed_path(target_path)

            self.assertFalse(result["valid"])
            self.assertTrue(
                any("course_title mismatch for course_calculus" in error for error in result["errors"])
            )
            self.assertTrue(
                any("nodes without relations: 孤立节点" in error for error in result["errors"])
            )
            self.assertEqual(result["metrics"]["isolated_node_count"], 1)

    def test_validate_seed_accepts_modern_ai_entity_types_in_course_batches(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            seed_path = root / "curated_seed_graph.json"
            batch_dir = root / "curated_batches"
            plan_path = root / "course_expansion_plan.json"
            batch_dir.mkdir()

            write_json(seed_path, {"source": "seed_root", "nodes": [], "relations": []})
            write_course_plan(
                plan_path,
                [
                    {
                        "source": "course_llm",
                        "course_title": "大模型",
                        "target_node_range": [3, 5],
                    }
                ],
            )

            target_path = batch_dir / "course_llm.json"
            write_json(
                target_path,
                {
                    "source": "course_llm",
                    "course_title": "大模型",
                    "nodes": [
                        {
                            "canonical_name": "DeepSeek-V3",
                            "entity_type": "model",
                            "description": "面向通用语言理解、推理和生成任务的大规模语言模型。",
                        },
                        {
                            "canonical_name": "vLLM",
                            "entity_type": "tool",
                            "description": "用于高吞吐大模型推理服务和调度的开源工具。",
                        },
                        {
                            "canonical_name": "LangChain",
                            "entity_type": "framework",
                            "description": "用于编排大模型应用、检索和工具调用的开发框架。",
                        },
                    ],
                    "relations": [
                        ["vLLM", "DeepSeek-V3", "RELATED_TO"],
                        ["LangChain", "DeepSeek-V3", "USES"],
                    ],
                },
            )

            with patch("txt2neo4j.validate_curated_batch.CURATED_SEED_PATH", seed_path), patch(
                "txt2neo4j.validate_curated_batch.CURATED_BATCH_DIR", batch_dir
            ), patch(
                "txt2neo4j.validate_curated_batch.COURSE_EXPANSION_PLAN_PATH", plan_path
            ):
                result = validate_seed_path(target_path)

            self.assertTrue(result["valid"])
            self.assertEqual(result["errors"], [])


if __name__ == "__main__":
    unittest.main()
