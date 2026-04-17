from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field


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


class FrontierMetadataAuthorInput(BaseModel):
    name: str
    affiliation: str = ""


class FrontierMetadataPaperInput(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    title: str
    abstract: str = ""
    keywords: list[str] = Field(default_factory=list)
    year: int | None = None
    venue: str = ""
    venue_type: str = Field(default="", alias="venueType")
    venue_rank: str = Field(default="", alias="venueRank")
    url: str = ""
    source: str = ""
    authors: list[str | FrontierMetadataAuthorInput] = Field(default_factory=list)
    updated_at: str | None = Field(default=None, alias="updatedAt")
    knowledge_hints: list[str] = Field(default_factory=list, alias="knowledgeHints")
    topic_hints: list[str] = Field(default_factory=list, alias="topicHints")


class FrontierMetadataInput(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    source: str = "frontier_metadata"
    generated_at: str | None = Field(default=None, alias="generatedAt")
    papers: list[FrontierMetadataPaperInput] = Field(default_factory=list)


class FrontierKnowledgeNodeRef(BaseModel):
    node_id: str
    knowledge_id: str
    name: str
    canonical_name: str
    entity_type: str = "concept"
    source_path: str = "frontier/generated"
    chapter_title: str = ""
    description: str = ""
    aliases: list[str] = Field(default_factory=list)


class FrontierPaper(BaseModel):
    paper_id: str
    title: str
    abstract: str = ""
    year: int | None = None
    venue: str = ""
    url: str = ""
    source: str = ""
    updated_at: str = ""
    keywords: list[str] = Field(default_factory=list)


class ResearchTopic(BaseModel):
    topic_id: str
    name: str
    canonical_name: str
    keywords: list[str] = Field(default_factory=list)
    summary: str = ""


class VenueNode(BaseModel):
    venue_id: str
    name: str
    type: str = ""
    rank: str = ""


class AuthorNode(BaseModel):
    author_id: str
    name: str
    affiliation: str = ""


class FrontierDigest(BaseModel):
    digest_id: str
    knowledge_id: str
    knowledge_name: str
    summary: str
    status: Literal["ready", "not_ready"] = "ready"
    generated_at: str


class PaperTopicLink(BaseModel):
    paper_id: str
    topic_id: str
    confidence: float = 1.0


class PaperVenueLink(BaseModel):
    paper_id: str
    venue_id: str


class PaperAuthorLink(BaseModel):
    paper_id: str
    author_id: str


class TopicKnowledgeLink(BaseModel):
    topic_id: str
    knowledge_node_id: str


class DigestKnowledgeLink(BaseModel):
    digest_id: str
    knowledge_node_id: str


class DigestPaperLink(BaseModel):
    digest_id: str
    paper_id: str


class FrontierBundle(BaseModel):
    source_path: str
    metadata_source: str
    generated_at: str
    knowledge_nodes: list[FrontierKnowledgeNodeRef] = Field(default_factory=list)
    papers: list[FrontierPaper] = Field(default_factory=list)
    topics: list[ResearchTopic] = Field(default_factory=list)
    venues: list[VenueNode] = Field(default_factory=list)
    authors: list[AuthorNode] = Field(default_factory=list)
    digests: list[FrontierDigest] = Field(default_factory=list)
    paper_topic_links: list[PaperTopicLink] = Field(default_factory=list)
    paper_venue_links: list[PaperVenueLink] = Field(default_factory=list)
    paper_author_links: list[PaperAuthorLink] = Field(default_factory=list)
    topic_knowledge_links: list[TopicKnowledgeLink] = Field(default_factory=list)
    digest_knowledge_links: list[DigestKnowledgeLink] = Field(default_factory=list)
    digest_paper_links: list[DigestPaperLink] = Field(default_factory=list)
    report: dict = Field(default_factory=dict)
