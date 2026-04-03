from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field


EntityType = Literal["topic", "concept", "method", "algorithm", "application", "exercise"]
StructureType = Literal["book", "chapter", "section", "subsection"]


class TextLine(BaseModel):
    line_no: int
    text: str


class StructureNode(BaseModel):
    node_id: str
    node_type: StructureType
    level: int
    code: str
    title: str
    canonical_name: str
    parent_id: str | None = None
    path: str
    line_no: int
    content: list[str] = Field(default_factory=list)


class KnowledgeNode(BaseModel):
    node_id: str
    name: str
    raw_name: str
    canonical_name: str
    entity_type: EntityType
    source_structure_id: str
    source_path: str
    chapter_title: str = ""
    description: str = ""
    aliases: list[str] = Field(default_factory=list)
    facets: list[str] = Field(default_factory=list)
    confidence: float = 0.0
    review_status: Literal["approved", "review"] = "approved"
    quality_issues: list[str] = Field(default_factory=list)
    source_line_no: int


class RelationRecord(BaseModel):
    relation_id: str
    source_id: str
    target_id: str
    relation_type: str
    confidence: float
    source_text: str = ""
    source_path: str = ""


class PipelineBundle(BaseModel):
    source_path: str
    cleaned_lines: list[TextLine]
    structure_nodes: list[StructureNode]
    knowledge_nodes: list[KnowledgeNode]
    relations: list[RelationRecord]
    quality_report: dict
