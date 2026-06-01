"""
构建外部知识库导入数据：
  - ACM CCS 2012 AI 子树（~82 概念）
  - AIO Ontology 精选概念（~150 ML/Model/Network 类）

输出两个 curated JSON 文件到 config/curated_batches/
"""
import json
import re
import csv
import os
from pathlib import Path

CONFIG_DIR = Path(__file__).resolve().parents[1] / "config" / "curated_batches"


def parse_acm_ccs(xml_path: str) -> list[dict]:
    """解析 ACM CCS SKOS XML，提取 AI 子树下所有概念。"""
    with open(xml_path, "r", encoding="utf-8") as f:
        content = f.read()

    # 解析所有 Concept
    concepts: dict[str, dict] = {}
    pattern = r'<skos:Concept rdf:about="#(\d+)"[^>]*>(.*?)</skos:Concept>'
    for m in re.finditer(pattern, content, re.DOTALL):
        cid = m.group(1)
        body = m.group(2)
        pref = re.search(r'<skos:prefLabel[^>]*>(.*?)</skos:prefLabel>', body)
        broader = re.findall(r'<skos:broader rdf:resource="#(\d+)"', body)
        narrower = re.findall(r'<skos:narrower rdf:resource="#(\d+)"', body)
        related = re.findall(r'<skos:related rdf:resource="#(\d+)"', body)
        alt_labels = re.findall(r'<skos:altLabel[^>]*>(.*?)</skos:altLabel>', body)
        concepts[cid] = {
            "label": pref.group(1) if pref else cid,
            "broader": broader,
            "narrower": narrower,
            "related": related,
            "altLabels": alt_labels,
        }

    # 找到 AI 根节点
    ai_root = None
    for cid, c in concepts.items():
        if c["label"].lower() == "artificial intelligence":
            ai_root = cid
            break
    if not ai_root:
        raise ValueError("ACM CCS 中未找到 Artificial intelligence 概念")

    # BFS 收集 AI 子树
    visited = set()
    queue = [ai_root]
    ai_ids = set()
    while queue:
        cid = queue.pop(0)
        if cid in visited:
            continue
        visited.add(cid)
        ai_ids.add(cid)
        for nid in concepts.get(cid, {}).get("narrower", []):
            if nid not in visited and nid in concepts:
                queue.append(nid)

    print(f"ACM CCS AI 子树: {len(ai_ids)} 个概念")

    # 构建节点列表（英文名作为 name，后续可补充中文翻译）
    nodes = []
    for cid in ai_ids:
        c = concepts[cid]
        label = c["label"]
        # HTML 实体解码
        label = label.replace("&#47;", "/").replace("&amp;", "&")
        nodes.append({
            "canonical_name": f"ACM:{label}",
            "entity_type": "topic",
            "description": f"{label} (ACM CCS 2012 分类)",
        })

    # 构建关系
    relations = []
    for cid in sorted(ai_ids):
        c = concepts[cid]
        label = c["label"].replace("&#47;", "/").replace("&amp;", "&")
        for nid in c.get("narrower", []):
            if nid in ai_ids:
                child_label = (
                    concepts[nid]["label"]
                    .replace("&#47;", "/")
                    .replace("&amp;", "&")
                )
                relations.append([
                    f"ACM:{label}",
                    f"ACM:{child_label}",
                    "SUBTOPIC_OF",
                ])
        for rid in c.get("related", []):
            if rid in ai_ids:
                rel_label = (
                    concepts[rid]["label"]
                    .replace("&#47;", "/")
                    .replace("&amp;", "&")
                )
                relations.append([
                    f"ACM:{label}",
                    f"ACM:{rel_label}",
                    "RELATED_TO",
                ])

    return nodes, relations


def parse_aio(csv_path: str) -> tuple[list[dict], list[list[str]]]:
    """解析 AIO CSV，提取 ML/Model/Network/Function 相关的教学适用概念。"""
    with open(csv_path, "r", encoding="utf-8") as f:
        lines = f.read().strip().split("\n")

    # 跳过两个 header 行
    classes = []
    for line in lines[2:]:
        parts = line.split(",")
        if len(parts) < 5:
            continue
        cid = parts[0]  # e.g., AIO:SoftmaxFunction
        subset = parts[1] if len(parts) > 1 else ""
        name = parts[4] if len(parts) > 4 else ""
        parent = parts[-1] if len(parts) > 1 else ""
        # description 在倒数第几列取决于CSV结构
        definition = parts[-5] if len(parts) >= 5 else ""
        classes.append({
            "id": cid,
            "subset": subset,
            "name": name,
            "parent": parent,
            "definition": definition,
        })

    print(f"AIO 总类数: {len(classes)}")

    # 筛选教学相关子集
    # Layer 子集太细（164个具体层），只保留有代表性的
    # 主要保留 ML/MODEL/NETWORK
    teaching_subsets = {
        "AIO:MachineLearningSubset",
        "AIO:ModelSubset",
        "AIO:NetworkSubset",
        "AIO:FunctionSubset",
        "AIO:FunctionSubset|AIO:ActivationFunctionSubset",
        "AIO:TrainingStrategySubset",
        "AIO:ClassSubset",
        "AIO:PreprocessingSubset",
    }

    selected = [c for c in classes if c["subset"] in teaching_subsets]
    # 也加入 Bias 子集的关键概念（AI 伦理/公平性相关）
    bias_selected = [
        c
        for c in classes
        if c["subset"] == "AIO:BiasSubset"
        and c["parent"] in ("AIO:Bias", "AIO:SystemicBias", "AIO:ComputationalBias")
    ]
    selected += bias_selected[:15]  # 限制数量

    print(f"精选 AIO 概念数: {len(selected)}")

    # 构建节点
    nodes = []
    aio_ids = set()
    for c in selected:
        short_name = c["name"].strip()
        aio_ids.add(c["id"])
        desc = c.get("definition", "").strip()
        if not desc or desc == short_name:
            desc = f"{short_name} (AIO Ontology)"
        nodes.append({
            "canonical_name": f"AIO:{short_name}",
            "entity_type": _map_entity_type(c["subset"]),
            "description": desc[:500],
        })

    # 构建关系（只保留两个节点都在 selected 中的）
    aio_id_set = {c["id"] for c in selected}
    id_to_name = {c["id"]: c["name"].strip() for c in selected}
    relations = []
    for c in selected:
        parent_id = c["parent"]
        # 跳过 owl:Thing 等根
        if parent_id.startswith("owl:") or parent_id.startswith("BFO:"):
            continue
        if parent_id in aio_id_set:
            relations.append([
                f'AIO:{id_to_name[c["id"]]}',
                f'AIO:{id_to_name[parent_id]}',
                "SUBTOPIC_OF",
            ])

    return nodes, relations


def _map_entity_type(subset: str) -> str:
    if "MachineLearning" in subset:
        return "method"
    if "Model" in subset:
        return "model"
    if "Network" in subset:
        return "framework"
    if "Function" in subset or "Activation" in subset:
        return "concept"
    if "Training" in subset:
        return "method"
    if "Preprocessing" in subset:
        return "method"
    if "Bias" in subset:
        return "concept"
    if "Class" in subset:
        return "topic"
    return "concept"


def build_and_save(
    acm_xml: str = "/tmp/acm_data/acm-ccs-skos.xml",
    aio_csv: str = "/tmp/aio_data/aio-src.csv",
    output_dir: str | None = None,
) -> tuple[Path, Path]:
    """构建并保存导入 JSON 文件。"""
    out = Path(output_dir) if output_dir else CONFIG_DIR
    out.mkdir(parents=True, exist_ok=True)

    # ACM CCS
    print("=== 解析 ACM CCS ===")
    acm_nodes, acm_relations = parse_acm_ccs(acm_xml)
    acm_bundle = {
        "source": "acm_ccs_2012_ai_subtree",
        "course_title": "ACM CCS 2012 AI 分类体系",
        "nodes": acm_nodes,
        "relations": acm_relations,
    }
    acm_path = out / "acm_ccs_ai_subtree.json"
    with open(acm_path, "w", encoding="utf-8") as f:
        json.dump(acm_bundle, f, ensure_ascii=False, indent=2)
    print(f"ACM CCS: {len(acm_nodes)} 节点, {len(acm_relations)} 关系 → {acm_path}")

    # AIO
    print("\n=== 解析 AIO ===")
    aio_nodes, aio_relations = parse_aio(aio_csv)
    aio_bundle = {
        "source": "aio_ontology_ml_core",
        "course_title": "AIO Ontology ML/DL 核心概念",
        "nodes": aio_nodes,
        "relations": aio_relations,
    }
    aio_path = out / "aio_ml_core.json"
    with open(aio_path, "w", encoding="utf-8") as f:
        json.dump(aio_bundle, f, ensure_ascii=False, indent=2)
    print(f"AIO: {len(aio_nodes)} 节点, {len(aio_relations)} 关系 → {aio_path}")

    return acm_path, aio_path


if __name__ == "__main__":
    build_and_save()
