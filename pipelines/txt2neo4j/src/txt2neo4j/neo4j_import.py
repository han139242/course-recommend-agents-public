from __future__ import annotations

from itertools import islice

from neo4j import GraphDatabase

from .models import FrontierBundle, PipelineBundle

SUPPORTED_RELATION_TYPES = {
    "PART_OF",
    "SUBTOPIC_OF",
    "USES",
    "APPLIES_TO",
    "PREREQUISITE_OF",
    "RELATED_TO",
}

FRONTIER_RELATION_SPECS = (
    ("MENTIONS_TOPIC", "Paper", "paper_id", "paper_id", "ResearchTopic", "topic_id", "topic_id"),
    ("PUBLISHED_IN", "Paper", "paper_id", "paper_id", "Venue", "venue_id", "venue_id"),
    ("AUTHORED_BY", "Paper", "paper_id", "paper_id", "Author", "author_id", "author_id"),
    ("EXTENDS_KNOWLEDGE", "ResearchTopic", "topic_id", "topic_id", "KnowledgeNode", "node_id", "knowledge_node_id"),
    ("CURATED_FOR", "FrontierDigest", "digest_id", "digest_id", "KnowledgeNode", "node_id", "knowledge_node_id"),
    ("SUMMARIZES_PAPER", "FrontierDigest", "digest_id", "digest_id", "Paper", "paper_id", "paper_id"),
)


def _batched(items, batch_size: int):
    iterator = iter(items)
    while True:
        chunk = list(islice(iterator, batch_size))
        if not chunk:
            break
        yield chunk


def create_constraints(driver) -> None:
    statements = [
        "CREATE CONSTRAINT knowledge_id_unique IF NOT EXISTS FOR (n:KnowledgeNode) REQUIRE n.node_id IS UNIQUE",
        "CREATE CONSTRAINT paper_id_unique IF NOT EXISTS FOR (n:Paper) REQUIRE n.paper_id IS UNIQUE",
        "CREATE CONSTRAINT topic_id_unique IF NOT EXISTS FOR (n:ResearchTopic) REQUIRE n.topic_id IS UNIQUE",
        "CREATE CONSTRAINT venue_id_unique IF NOT EXISTS FOR (n:Venue) REQUIRE n.venue_id IS UNIQUE",
        "CREATE CONSTRAINT author_id_unique IF NOT EXISTS FOR (n:Author) REQUIRE n.author_id IS UNIQUE",
        "CREATE CONSTRAINT digest_id_unique IF NOT EXISTS FOR (n:FrontierDigest) REQUIRE n.digest_id IS UNIQUE",
    ]
    with driver.session() as session:
        for statement in statements:
            session.run(statement)


def import_bundle(
    bundle: PipelineBundle,
    uri: str,
    username: str,
    password: str,
    database: str = "neo4j",
    batch_size: int = 200,
) -> None:
    driver = GraphDatabase.driver(uri, auth=(username, password))

    try:
        create_constraints(driver)

        with driver.session(database=database) as session:
            for batch in _batched(bundle.knowledge_nodes, batch_size):
                session.run(
                    """
                    UNWIND $rows AS row
                    MERGE (n:KnowledgeNode {node_id: row.node_id})
                    SET n += row
                    """,
                    rows=[node.model_dump() for node in batch],
                )

            for relation_type in SUPPORTED_RELATION_TYPES:
                typed_rows = [
                    relation.model_dump()
                    for relation in bundle.relations
                    if relation.relation_type == relation_type
                ]
                if not typed_rows:
                    continue

                query = f"""
                UNWIND $rows AS row
                MATCH (a:KnowledgeNode {{node_id: row.source_id}})
                MATCH (b:KnowledgeNode {{node_id: row.target_id}})
                MERGE (a)-[r:{relation_type}]->(b)
                SET r += row
                """

                for batch in _batched(typed_rows, batch_size):
                    session.run(query, rows=batch)
    finally:
        driver.close()


def _import_rows(session, label: str, id_field: str, rows: list[dict], batch_size: int) -> None:
    if not rows:
        return
    query = f"""
    UNWIND $rows AS row
    MERGE (n:{label} {{{id_field}: row.{id_field}}})
    SET n += row
    """
    for batch in _batched(rows, batch_size):
        session.run(query, rows=batch)


def _import_frontier_knowledge_rows(session, rows: list[dict], batch_size: int) -> None:
    if not rows:
        return
    query = """
    UNWIND $rows AS row
    MERGE (n:KnowledgeNode {node_id: row.node_id})
    ON CREATE SET
        n.knowledge_id = row.knowledge_id,
        n.name = row.name,
        n.canonical_name = row.canonical_name,
        n.entity_type = row.entity_type,
        n.source_path = row.source_path,
        n.chapter_title = row.chapter_title,
        n.description = row.description,
        n.aliases = row.aliases,
        n.frontier_seeded = true,
        n.source_kind = 'frontier_seed'
    SET
        n.knowledge_id = coalesce(n.knowledge_id, row.knowledge_id),
        n.name = coalesce(n.name, row.name),
        n.canonical_name = coalesce(n.canonical_name, row.canonical_name),
        n.entity_type = coalesce(n.entity_type, row.entity_type),
        n.source_path = coalesce(n.source_path, row.source_path),
        n.chapter_title = coalesce(n.chapter_title, row.chapter_title),
        n.description = CASE
            WHEN coalesce(n.description, '') = '' THEN row.description
            ELSE n.description
        END,
        n.aliases = CASE
            WHEN size(coalesce(n.aliases, [])) = 0 THEN row.aliases
            ELSE n.aliases
        END,
        n.frontier_seeded = true
    """
    for batch in _batched(rows, batch_size):
        session.run(query, rows=batch)


def _import_frontier_relation_rows(
    session,
    relation_type: str,
    from_label: str,
    from_node_key: str,
    from_row_key: str,
    to_label: str,
    to_node_key: str,
    to_row_key: str,
    rows: list[dict],
    batch_size: int,
) -> None:
    if not rows:
        return
    query = f"""
    UNWIND $rows AS row
    MATCH (a:{from_label} {{{from_node_key}: row.{from_row_key}}})
    MATCH (b:{to_label} {{{to_node_key}: row.{to_row_key}}})
    MERGE (a)-[r:{relation_type}]->(b)
    SET r += row
    """
    for batch in _batched(rows, batch_size):
        session.run(query, rows=batch)


def import_frontier_bundle(
    bundle: FrontierBundle,
    uri: str,
    username: str,
    password: str,
    database: str = "neo4j",
    batch_size: int = 200,
) -> None:
    driver = GraphDatabase.driver(uri, auth=(username, password))
    try:
        create_constraints(driver)
        with driver.session(database=database) as session:
            _import_frontier_knowledge_rows(
                session,
                [node.model_dump() for node in bundle.knowledge_nodes],
                batch_size,
            )
            _import_rows(
                session,
                label="Paper",
                id_field="paper_id",
                rows=[paper.model_dump() for paper in bundle.papers],
                batch_size=batch_size,
            )
            _import_rows(
                session,
                label="ResearchTopic",
                id_field="topic_id",
                rows=[topic.model_dump() for topic in bundle.topics],
                batch_size=batch_size,
            )
            _import_rows(
                session,
                label="Venue",
                id_field="venue_id",
                rows=[venue.model_dump() for venue in bundle.venues],
                batch_size=batch_size,
            )
            _import_rows(
                session,
                label="Author",
                id_field="author_id",
                rows=[author.model_dump() for author in bundle.authors],
                batch_size=batch_size,
            )
            _import_rows(
                session,
                label="FrontierDigest",
                id_field="digest_id",
                rows=[digest.model_dump() for digest in bundle.digests],
                batch_size=batch_size,
            )

            relation_row_map = {
                "MENTIONS_TOPIC": [row.model_dump() for row in bundle.paper_topic_links],
                "PUBLISHED_IN": [row.model_dump() for row in bundle.paper_venue_links],
                "AUTHORED_BY": [row.model_dump() for row in bundle.paper_author_links],
                "EXTENDS_KNOWLEDGE": [row.model_dump() for row in bundle.topic_knowledge_links],
                "CURATED_FOR": [row.model_dump() for row in bundle.digest_knowledge_links],
                "SUMMARIZES_PAPER": [row.model_dump() for row in bundle.digest_paper_links],
            }
            for relation_type, from_label, from_node_key, from_row_key, to_label, to_node_key, to_row_key in FRONTIER_RELATION_SPECS:
                _import_frontier_relation_rows(
                    session=session,
                    relation_type=relation_type,
                    from_label=from_label,
                    from_node_key=from_node_key,
                    from_row_key=from_row_key,
                    to_label=to_label,
                    to_node_key=to_node_key,
                    to_row_key=to_row_key,
                    rows=relation_row_map[relation_type],
                    batch_size=batch_size,
                )
    finally:
        driver.close()
