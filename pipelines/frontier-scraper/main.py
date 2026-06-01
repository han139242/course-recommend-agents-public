from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from datetime import UTC, datetime
from pathlib import Path

SCRAPER_ROOT = Path(__file__).resolve().parent
REPO_ROOT = SCRAPER_ROOT.parents[1]
TXT2NEO4J_ROOT = REPO_ROOT / "pipelines" / "txt2neo4j"
DEFAULT_OUTPUT = TXT2NEO4J_ROOT / "config" / "frontier_seed_metadata.json"
DEFAULT_FRONTIER_OUTPUT_DIR = TXT2NEO4J_ROOT / "artifacts" / "frontier"

sys.path.insert(0, str(SCRAPER_ROOT))

from ai_summarizer import enrich_papers_batch, map_papers_to_knowledge_batch  # noqa: E402
from arxiv_fetcher import AI_CATEGORIES, dedupe_key, fetch_all_categories, papers_to_dicts  # noqa: E402
from github_fetcher import fetch_github_releases  # noqa: E402
from huggingface_fetcher import fetch_huggingface_daily_papers  # noqa: E402
from sources import DEFAULT_GITHUB_FEEDS  # noqa: E402


def now_iso() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def load_existing_papers(path: Path) -> list[dict]:
    if not path.exists():
        return []
    payload = json.loads(path.read_text(encoding="utf-8"))
    if isinstance(payload, list):
        return [item for item in payload if isinstance(item, dict)]
    if isinstance(payload, dict):
        papers = payload.get("papers", [])
        return [item for item in papers if isinstance(item, dict)]
    return []


def merge_papers(new_papers: list[dict], existing_papers: list[dict], keep_existing: bool) -> list[dict]:
    merged: list[dict] = []
    seen: set[str] = set()

    for paper in new_papers:
        key = dedupe_key(paper)
        if key in seen:
            continue
        seen.add(key)
        merged.append(paper)

    if keep_existing:
        for paper in existing_papers:
            key = dedupe_key(paper)
            if key in seen:
                continue
            seen.add(key)
            merged.append(paper)

    return merged


def write_metadata(path: Path, papers: list[dict], source_names: list[str]) -> dict:
    payload = {
        "source": "frontier_sync",
        "generatedAt": now_iso(),
        "sources": source_names,
        "papers": papers,
    }
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    return payload


def run_txt2neo4j(frontier_input: Path, output_dir: Path, import_neo4j: bool = False) -> dict:
    command = [
        sys.executable,
        "-m",
        "txt2neo4j.cli",
        "--frontier-input",
        str(frontier_input.resolve()),
        "--frontier-output-dir",
        str(output_dir.resolve()),
    ]
    if import_neo4j:
        command.append("--import-frontier-neo4j")

    env = os.environ.copy()
    src_root = TXT2NEO4J_ROOT / "src"
    env["PYTHONPATH"] = str(src_root) + os.pathsep + env.get("PYTHONPATH", "")
    completed = subprocess.run(
        command,
        cwd=str(TXT2NEO4J_ROOT),
        env=env,
        check=True,
        text=True,
        capture_output=True,
    )
    stdout = completed.stdout.strip()
    try:
        return json.loads(stdout)
    except json.JSONDecodeError:
        return {"stdout": stdout, "stderr": completed.stderr.strip()}


def collect_sources(args: argparse.Namespace) -> tuple[list[dict], list[str]]:
    papers: list[dict] = []
    source_names: list[str] = []

    if "arxiv" in args.source:
        print("[1/4] 正在从 arXiv 官方 API 拉取最新论文...")
        fetched = fetch_all_categories(
            categories=args.categories or list(AI_CATEGORIES.keys()),
            max_per_category=args.max_per_category,
            query=args.query,
            since_days=args.since_days,
            request_delay=args.arxiv_delay,
        )
        papers.extend(papers_to_dicts(fetched))
        source_names.append("arXiv")
        print(f"      arXiv 收录 {len(fetched)} 篇")

    if "huggingface" in args.source:
        print("[2/4] 正在从 Hugging Face Daily Papers 拉取社区论文趋势...")
        try:
            fetched = fetch_huggingface_daily_papers(limit=args.hf_limit)
            papers.extend(fetched)
            source_names.append("Hugging Face Daily Papers")
            print(f"      Hugging Face 收录 {len(fetched)} 条")
        except Exception as exc:
            print(f"      Hugging Face 源暂不可用：{exc}")

    if "github" in args.source:
        print("[3/4] 正在检查 GitHub 开源前沿观察源 releases...")
        token = os.getenv("GITHUB_TOKEN", "")
        fetched = fetch_github_releases(
            sources=DEFAULT_GITHUB_FEEDS,
            limit=args.github_limit,
            since_days=args.since_days,
            token=token,
        )
        papers.extend(fetched)
        source_names.append("GitHub Releases")
        print(f"      GitHub 收录 {len(fetched)} 条")

    return papers, source_names


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="AI 前沿信息同步器：公开论文源 / GitHub 观察源 -> frontier metadata -> txt2neo4j artifacts",
    )
    parser.add_argument(
        "--source",
        nargs="+",
        choices=["arxiv", "huggingface", "github"],
        default=["arxiv"],
        help="同步源。默认 arxiv；可组合 arxiv huggingface github。",
    )
    parser.add_argument("--categories", nargs="*", default=None, help="arXiv 分类，例如 cs.AI cs.LG cs.CL。")
    parser.add_argument("--query", default="", help="arXiv all: 关键词过滤。")
    parser.add_argument("--since-days", type=int, default=14, help="仅同步最近 N 天内容；0 表示不加日期窗口。")
    parser.add_argument("--max-per-category", type=int, default=5, help="每个 arXiv 分类最多抓取论文数。")
    parser.add_argument("--hf-limit", type=int, default=20, help="Hugging Face Daily Papers 最大条数。")
    parser.add_argument("--github-limit", type=int, default=10, help="每个 GitHub 仓库 releases 最大条数。")
    parser.add_argument("--arxiv-delay", type=float, default=3.1, help="arXiv 连续请求间隔秒数。")
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT, help="frontier metadata JSON 输出路径。")
    parser.add_argument("--keep-existing", action="store_true", help="保留输出文件中已有论文并与新数据去重合并。")
    parser.add_argument("--skip-ai", action="store_true", help="跳过全部 DeepSeek/OpenAI 增强。")
    parser.add_argument("--skip-ai-summary", action="store_true", help="跳过 AI 摘要，只保留 AI 知识点映射。")
    parser.add_argument("--skip-ai-mapping", action="store_true", help="跳过 AI 知识点候选映射，只使用规则映射。")
    parser.add_argument("--ai-limit", type=int, default=None, help="最多增强前 N 篇论文，空值表示全部。")
    parser.add_argument("--ai-delay", type=float, default=1.0, help="AI 摘要请求间隔秒数。")
    parser.add_argument("--model", default=None, help="AI 模型名，默认读 DEEPSEEK_MODEL 或 deepseek-chat。")
    parser.add_argument(
        "--run-txt2neo4j",
        action="store_true",
        help="生成 metadata 后立即调用 txt2neo4j 生成前沿 JSON/CSV 产物，不写数据库。",
    )
    parser.add_argument(
        "--frontier-output-dir",
        type=Path,
        default=DEFAULT_FRONTIER_OUTPUT_DIR,
        help="txt2neo4j 前沿产物目录。",
    )
    parser.add_argument(
        "--import-neo4j",
        action="store_true",
        help="危险操作：在 --run-txt2neo4j 后继续导入 Neo4j。默认关闭。",
    )
    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()

    if args.import_neo4j and not args.run_txt2neo4j:
        parser.error("--import-neo4j 必须与 --run-txt2neo4j 一起使用。")

    if args.import_neo4j and os.getenv("FRONTIER_IMPORT_NEO4J_CONFIRMED") != "true":
        parser.error("导入 Neo4j 是高风险写库操作。请先确认风险，并设置 FRONTIER_IMPORT_NEO4J_CONFIRMED=true 后重试。")

    print("AI 前沿信息同步开始")
    print(f"输出 metadata: {args.output}")
    print(f"数据源: {', '.join(args.source)}")
    print(f"日期窗口: {'不限' if args.since_days <= 0 else f'最近 {args.since_days} 天'}")

    new_papers, source_names = collect_sources(args)
    if not new_papers:
        raise SystemExit("未从任何源获取到可用内容。")

    existing = load_existing_papers(args.output) if args.keep_existing else []
    merged = merge_papers(new_papers, existing, keep_existing=args.keep_existing)
    print(f"[4/4] 去重完成：新增候选 {len(new_papers)} 条，最终输出 {len(merged)} 条")

    has_ai_key = bool(os.getenv("DEEPSEEK_API_KEY") or os.getenv("OPENAI_API_KEY"))
    ai_disabled = args.skip_ai or not has_ai_key
    if ai_disabled and not args.skip_ai and not has_ai_key:
        print("未检测到 DEEPSEEK_API_KEY/OPENAI_API_KEY，自动跳过 AI 增强，使用规则映射。")

    if not ai_disabled and not args.skip_ai_summary:
        print("正在进行 AI 摘要增强...")
        merged = enrich_papers_batch(merged, delay=args.ai_delay, model=args.model, limit=args.ai_limit)
    else:
        print("已跳过 AI 摘要增强。")

    if not ai_disabled and not args.skip_ai_mapping:
        print("正在基于课程知识点候选集进行 AI 映射...")
        merged = map_papers_to_knowledge_batch(merged, delay=args.ai_delay, model=args.model, limit=args.ai_limit)
    else:
        print("已跳过 AI 知识点映射，使用规则提取的 knowledgeHints/topicHints。")

    payload = write_metadata(args.output, merged, source_names)
    print(f"已写出 metadata: {args.output.resolve()}")
    print(f"论文/前沿条目数: {len(payload['papers'])}")

    if args.run_txt2neo4j:
        print("正在生成 txt2neo4j 前沿产物...")
        report = run_txt2neo4j(args.output, args.frontier_output_dir, import_neo4j=args.import_neo4j)
        print(json.dumps(report, ensure_ascii=False, indent=2))
    else:
        print("下一步可运行：")
        print(
            f"  python main.py --skip-ai --run-txt2neo4j --output \"{args.output}\" "
            f"--frontier-output-dir \"{args.frontier_output_dir}\""
        )


if __name__ == "__main__":
    main()
