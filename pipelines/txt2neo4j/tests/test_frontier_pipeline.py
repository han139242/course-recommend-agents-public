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

from txt2neo4j.frontier_pipeline import run_frontier_pipeline  # noqa: E402


class FrontierPipelineTest(unittest.TestCase):
    def test_seed_metadata_generates_frontier_bundle(self) -> None:
        input_path = PROJECT_ROOT / "config" / "frontier_seed_metadata.json"
        with tempfile.TemporaryDirectory() as temp_dir:
            bundle = run_frontier_pipeline(input_path, Path(temp_dir))

            self.assertGreaterEqual(len(bundle.papers), 5)
            self.assertGreaterEqual(len(bundle.topics), 5)
            self.assertGreaterEqual(len(bundle.digests), 4)
            self.assertEqual(bundle.report["unmatched_papers"], 0)
            self.assertTrue(any(item.name == "Transformer" for item in bundle.knowledge_nodes))
            self.assertTrue(any(item.knowledge_name == "检索增强生成" for item in bundle.digests))
            self.assertTrue(all(item.status == "ready" for item in bundle.digests))
            self.assertGreater(len(bundle.paper_topic_links), 0)
            self.assertGreater(len(bundle.topic_knowledge_links), 0)
            self.assertGreater(len(bundle.digest_paper_links), 0)

            self.assertTrue((Path(temp_dir) / "papers.json").exists())
            self.assertTrue((Path(temp_dir) / "report.json").exists())
            self.assertTrue((Path(temp_dir) / "neo4j" / "papers.csv").exists())

    def test_unknown_hint_falls_back_to_placeholder_knowledge_node(self) -> None:
        payload = {
            "source": "frontier_placeholder_test",
            "generatedAt": "2026-04-17T12:00:00Z",
            "papers": [
                {
                    "title": "Fractal Cache Synthesis for Long-Horizon Control",
                    "abstract": "A lightweight controller composes latent cache fragments for stable long-horizon decision making.",
                    "keywords": ["fractal cache", "latent routing"],
                    "year": 2026,
                    "venue": "TestConf",
                    "knowledgeHints": ["Quantum Memory Router"],
                    "topicHints": ["Long-horizon cache routing"],
                }
            ],
        }

        with tempfile.TemporaryDirectory() as temp_dir:
            input_path = Path(temp_dir) / "placeholder.json"
            input_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")

            bundle = run_frontier_pipeline(input_path, Path(temp_dir) / "out")

            self.assertEqual(len(bundle.papers), 1)
            self.assertEqual(len(bundle.digests), 1)
            self.assertTrue(any(item.canonical_name == "Quantum Memory Router" for item in bundle.knowledge_nodes))


if __name__ == "__main__":
    unittest.main()
