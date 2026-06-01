from __future__ import annotations

import sys
import unittest
from pathlib import Path

SCRAPER_ROOT = Path(__file__).resolve().parents[1]
if str(SCRAPER_ROOT) not in sys.path:
    sys.path.insert(0, str(SCRAPER_ROOT))

from arxiv_fetcher import build_search_query, dedupe_key, papers_to_dicts, parse_arxiv_feed  # noqa: E402


SAMPLE_FEED = """<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:arxiv="http://arxiv.org/schemas/atom">
  <entry>
    <id>http://arxiv.org/abs/2605.12345v2</id>
    <updated>2026-05-31T10:00:00Z</updated>
    <published>2026-05-30T10:00:00Z</published>
    <title>Retrieval-Augmented Transformer Agents for Knowledge Graph Reasoning</title>
    <summary>
      We study large language model agents that combine retrieval augmented generation,
      knowledge graph reasoning, and transformer planning for reliable tool use.
    </summary>
    <author><name>Ada Chen</name></author>
    <author><name>Bo Liu</name></author>
    <arxiv:primary_category term="cs.AI" scheme="http://arxiv.org/schemas/atom"/>
    <category term="cs.AI" scheme="http://arxiv.org/schemas/atom"/>
    <category term="cs.CL" scheme="http://arxiv.org/schemas/atom"/>
  </entry>
</feed>
"""


class ArxivFetcherTest(unittest.TestCase):
    def test_parse_arxiv_feed_normalizes_frontier_metadata(self) -> None:
        papers = parse_arxiv_feed(SAMPLE_FEED, requested_category="cs.AI")

        self.assertEqual(len(papers), 1)
        paper = papers[0]
        self.assertEqual(paper.source_id, "2605.12345")
        self.assertEqual(paper.year, 2026)
        self.assertEqual(paper.url, "https://arxiv.org/abs/2605.12345")
        self.assertEqual([item["name"] for item in paper.authors], ["Ada Chen", "Bo Liu"])
        self.assertIn("Retrieval-Augmented Generation", paper.keywords)
        self.assertIn("检索增强生成", paper.knowledge_hints)
        self.assertIn("人工智能", paper.knowledge_hints)

        payload = papers_to_dicts(papers)[0]
        self.assertEqual(payload["venueType"], "preprint")
        self.assertEqual(payload["sourceName"], "arXiv")
        self.assertEqual(payload["category"], "cs.AI")

    def test_build_search_query_supports_date_window(self) -> None:
        query = build_search_query("cs.LG", '"large language model"', since_days=7)

        self.assertIn("cat:cs.LG", query)
        self.assertIn('all:"large language model"', query)
        self.assertIn("submittedDate:[", query)

    def test_dedupe_key_prefers_source_id(self) -> None:
        key = dedupe_key({"sourceId": "2605.12345", "url": "https://example.com/a", "title": "A"})

        self.assertEqual(key, "id:2605.12345")


if __name__ == "__main__":
    unittest.main()
