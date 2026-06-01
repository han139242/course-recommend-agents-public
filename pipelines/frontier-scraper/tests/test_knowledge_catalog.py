from __future__ import annotations

import sys
import unittest
from pathlib import Path

SCRAPER_ROOT = Path(__file__).resolve().parents[1]
if str(SCRAPER_ROOT) not in sys.path:
    sys.path.insert(0, str(SCRAPER_ROOT))

from knowledge_catalog import KnowledgeCandidate, candidates_to_prompt, select_knowledge_candidates  # noqa: E402


class KnowledgeCatalogTest(unittest.TestCase):
    def test_select_candidates_prefers_text_and_seed_hints(self) -> None:
        catalog = [
            KnowledgeCandidate(name="Transformer", course="大模型基础", aliases=("Transformer",)),
            KnowledgeCandidate(name="知识图谱", course="知识工程", aliases=("Knowledge Graph", "知识图谱")),
            KnowledgeCandidate(name="强化学习", course="机器学习", aliases=("Reinforcement Learning",)),
        ]

        selected = select_knowledge_candidates(
            title="Retrieval-Augmented Transformer Agents for Knowledge Graph Reasoning",
            abstract="The model combines transformer planning and knowledge graph reasoning.",
            seed_hints=["知识图谱"],
            catalog=catalog,
        )

        self.assertEqual([item.name for item in selected[:2]], ["知识图谱", "Transformer"])

    def test_candidates_to_prompt_contains_course_context(self) -> None:
        prompt = candidates_to_prompt([
            KnowledgeCandidate(name="检索增强生成", course="大模型应用", aliases=("RAG",), description="结合检索和生成。")
        ])

        self.assertIn("检索增强生成", prompt)
        self.assertIn("大模型应用", prompt)
        self.assertIn("RAG", prompt)


if __name__ == "__main__":
    unittest.main()
