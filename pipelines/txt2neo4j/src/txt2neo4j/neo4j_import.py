from __future__ import annotations

from itertools import islice

from neo4j import GraphDatabase

from .models import PipelineBundle

SUPPORTED_RELATION_TYPES = {
    "PART_OF",
    "SUBTOPIC_OF",
    "USES",
    "APPLIES_TO",
    "PREREQUISITE_OF",
    "RELATED_TO",
}


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
