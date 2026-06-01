"""
将 ACM CCS 和 AIO curated JSON 文件直接Importing Neo4j。
遵循 curated_graph.py 的 merge 逻辑：按 canonical_name 去重，不重复创建已存在nodes。

用法：
  python scripts/import_external_knowledge.py [--acm] [--aio] [--dry-run]
"""
import json
import os
import sys
from hashlib import sha1
from pathlib import Path

from dotenv import load_dotenv

# Loading .env - 使用绝对路径避免中文编码问题
import glob as _glob
_env_path = None
for _p in [
    "C:/Users/USER/Desktop/python-word/大创/course-recommend-agents-xiangmu/.env",
]:
    if os.path.isfile(_p):
        _env_path = _p
        break
if _env_path:
    load_dotenv(_env_path)
    print(f"Loading .env: {_env_path}")
else:
    # fallback: try relative from project root
    _proj_root = Path(__file__).resolve().parents[2]
    _env_candidate = _proj_root / ".env"
    if _env_candidate.exists():
        load_dotenv(_env_candidate)
        print(f"Loading .env (fallback): {_env_candidate}")
    else:
        print("WARNING: .env not found, using env vars")


def _stable_id(*parts: str) -> str:
    return sha1("||".join(parts).encode("utf-8")).hexdigest()[:16]


def connect_neo4j():
    from neo4j import GraphDatabase, basic_auth

    uri = os.getenv("NEO4J_URI", "bolt://127.0.0.1:7687")
    username = os.getenv("NEO4J_USER", os.getenv("NEO4J_USERNAME", "neo4j"))
    password = os.getenv("NEO4J_PASSWORD", "")
    database = os.getenv("NEO4J_DATABASE", "neo4j")
    print(f"Connecting Neo4j: {uri}, 用户: {username}, 数据库: {database}")
    driver = GraphDatabase.driver(uri, auth=basic_auth(username, password))
    return driver, database


SUPPORTED_RELATION_TYPES = {
    "PART_OF",
    "SUBTOPIC_OF",
    "USES",
    "APPLIES_TO",
    "PREREQUISITE_OF",
    "RELATED_TO",
}


def import_bundle(driver, database: str, bundle: dict, dry_run: bool = False):
    """Importing一个 curated bundle 到 Neo4j。"""
    nodes = bundle.get("nodes", [])
    relations = bundle.get("relations", [])
    source = bundle.get("source", "external_knowledge")
    course_title = bundle.get("course_title", "External Knowledge")

    print(f"\n{'[DRY RUN] ' if dry_run else ''}Importing {source}:")
    print(f"  nodes: {len(nodes)}")
    print(f"  relations: {len(relations)}")

    if dry_run:
        return

    # Step 1: 加载已存在的nodes（按 canonical_name）
    with driver.session(database=database) as session:
        existing = {}
        result = session.run(
            "MATCH (n:KnowledgeNode) WHERE n.canonical_name IS NOT NULL "
            "RETURN n.canonical_name AS name, n.node_id AS id"
        )
        for record in result:
            existing[record["name"]] = record["id"]

    created = 0
    merged = 0

    # Step 2: 创建/合并nodes
    with driver.session(database=database) as session:
        for node in nodes:
            canonical = node["canonical_name"]
            entity_type = node.get("entity_type", "concept")
            description = node.get("description", "")

            if canonical in existing:
                # 已有nodes → 补充描述（如果缺失）
                session.run(
                    "MATCH (n:KnowledgeNode {canonical_name: $cn}) "
                    "WHERE n.description IS NULL OR n.description = '' "
                    "SET n.description = $desc",
                    cn=canonical,
                    desc=description,
                )
                merged += 1
            else:
                node_id = _stable_id("external", source, canonical)
                session.run(
                    "CREATE (n:KnowledgeNode {"
                    "  node_id: $nid,"
                    "  canonical_name: $cn,"
                    "  name: $name,"
                    "  entity_type: $et,"
                    "  description: $desc,"
                    "  source_kind: 'curated',"
                    "  source_path: $sp,"
                    "  quality_tier: 'silver',"
                    "  review_status: 'review',"
                    "  graph_namespace: 'ai_course_kg',"
                    "  data_standard: 'industrial_v1',"
                    "  curated_source: $cs"
                    "})",
                    nid=node_id,
                    cn=canonical,
                    name=canonical,
                    et=entity_type,
                    desc=description,
                    sp=f"external/{source}",
                    cs=source,
                )
                existing[canonical] = node_id
                created += 1

    print(f"  Created: {created}, Existing (skipped): {merged}")

    # Step 3: 创建relations
    rel_created = 0
    rel_skipped = 0
    with driver.session(database=database) as session:
        for rel in relations:
            if len(rel) != 3:
                continue
            src_name, tgt_name, rel_type = rel[0], rel[1], rel[2]

            if rel_type not in SUPPORTED_RELATION_TYPES:
                rel_type = "RELATED_TO"

            src_id = existing.get(src_name)
            tgt_id = existing.get(tgt_name)
            if not src_id or not tgt_id:
                rel_skipped += 1
                continue

            rel_id = _stable_id("ext_rel", src_id, tgt_id, rel_type)
            session.run(
                f"MATCH (a:KnowledgeNode {{node_id: $src}}), "
                f"(b:KnowledgeNode {{node_id: $tgt}}) "
                f"MERGE (a)-[r:{rel_type} {{relation_id: $rid}}]->(b) "
                f"SET r.confidence = 0.6, r.relation_source = $rs",
                src=src_id,
                tgt=tgt_id,
                rid=rel_id,
                rs="external_knowledge",
            )
            rel_created += 1

    print(f"  Relations created: {rel_created}, skipped: {rel_skipped}")
    return {"created": created, "merged": merged, "relations": rel_created}


def main():
    dry_run = "--dry-run" in sys.argv
    acm_flag = "--acm" in sys.argv
    aio_flag = "--aio" in sys.argv

    # 默认全Importing
    if not acm_flag and not aio_flag:
        acm_flag = aio_flag = True

    config_dir = Path("C:/Users/USER/Desktop/python-word/大创/course-recommend-agents-xiangmu/pipelines/txt2neo4j/config/curated_batches")
    acm_path = config_dir / "acm_ccs_ai_subtree.json"
    aio_path = config_dir / "aio_ml_core.json"

    driver, database = connect_neo4j()
    try:
        if acm_flag and acm_path.exists():
            bundle = json.loads(acm_path.read_text(encoding="utf-8"))
            import_bundle(driver, database, bundle, dry_run)

        if aio_flag and aio_path.exists():
            bundle = json.loads(aio_path.read_text(encoding="utf-8"))
            import_bundle(driver, database, bundle, dry_run)

        # 验证结果
        if not dry_run:
            with driver.session(database=database) as session:
                total = session.run(
                    "MATCH (n:KnowledgeNode) RETURN count(n) AS cnt"
                ).single()["cnt"]
                rels = session.run(
                    "MATCH ()-[r]->() RETURN count(r) AS cnt"
                ).single()["cnt"]
                external = session.run(
                    "MATCH (n:KnowledgeNode) "
                    "WHERE n.curated_source IN ['acm_ccs_2012_ai_subtree', 'aio_ontology_ml_core'] "
                    "RETURN count(n) AS cnt"
                ).single()["cnt"]
                print(f"\n=== Importing完成 ===")
                print(f"KnowledgeNode total: {total}")
                print(f"relationstotal: {rels}")
                print(f"New external knowledge nodes: {external}")
    finally:
        driver.close()


if __name__ == "__main__":
    main()
