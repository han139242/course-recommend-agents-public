from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

SCRAPER_ROOT = Path(__file__).resolve().parents[1]
if str(SCRAPER_ROOT) not in sys.path:
    sys.path.insert(0, str(SCRAPER_ROOT))

from main import load_existing_papers, merge_papers, write_metadata  # noqa: E402
from sources import github_repo_api_url  # noqa: E402


class FrontierSyncTest(unittest.TestCase):
    def test_merge_papers_deduplicates_new_then_preserves_existing(self) -> None:
        new_papers = [
            {"title": "A", "sourceId": "x", "url": "https://example.com/a"},
            {"title": "A duplicate", "sourceId": "x", "url": "https://example.com/a2"},
        ]
        existing = [
            {"title": "Old", "sourceId": "old", "url": "https://example.com/old"},
            {"title": "Old duplicate", "sourceId": "x", "url": "https://example.com/a"},
        ]

        merged = merge_papers(new_papers, existing, keep_existing=True)

        self.assertEqual([item["sourceId"] for item in merged], ["x", "old"])

    def test_write_and_load_metadata_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            path = Path(temp_dir) / "frontier.json"
            write_metadata(path, [{"title": "Paper", "sourceId": "p1"}], ["arXiv"])

            payload = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(payload["source"], "frontier_sync")
            self.assertEqual(payload["sources"], ["arXiv"])
            self.assertEqual(load_existing_papers(path)[0]["sourceId"], "p1")

    def test_github_repo_api_url(self) -> None:
        url = github_repo_api_url("https://github.com/paperswithcode/paperswithcode-data")

        self.assertEqual(url, "https://api.github.com/repos/paperswithcode/paperswithcode-data/releases")


if __name__ == "__main__":
    unittest.main()
