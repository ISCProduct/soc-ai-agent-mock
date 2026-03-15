"""
グラフ出力モジュール

GraphML と JSON（nodes/edges 形式）の両形式で出力する。
"""
from __future__ import annotations

import json
import logging
import xml.etree.ElementTree as ET
from pathlib import Path

from models import CompanyNode, RelationEdge

logger = logging.getLogger(__name__)


# ------------------------------------------------------------------ #
# JSON
# ------------------------------------------------------------------ #

def _node_to_dict(node: CompanyNode) -> dict:
    return {
        "id": node.corporate_number,
        "official_name": node.official_name,
        "source_urls": node.source_urls,
        "business_category": node.business_category,
        "address": node.address,
        "website": node.website,
        "capital": node.capital,
        "employees": node.employees,
        "match_score": round(node.match_score, 4),
        "needs_review": node.needs_review,
    }


def _edge_to_dict(edge: RelationEdge) -> dict:
    return {
        "source": edge.source_corporate_number,
        "target": edge.target_corporate_number,
        "target_raw_name": edge.target_raw_name,
        "relation_type": edge.relation_type,
        "match_score": round(edge.match_score, 4),
        "needs_review": edge.needs_review,
    }


def write_json(
    nodes: dict[str, CompanyNode],
    edges: list[RelationEdge],
    output_path: Path,
) -> None:
    data = {
        "nodes": [_node_to_dict(n) for n in nodes.values()],
        "edges": [_edge_to_dict(e) for e in edges],
    }
    output_path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
    logger.info("JSON written: %s (%d nodes, %d edges)", output_path, len(nodes), len(edges))


# ------------------------------------------------------------------ #
# GraphML
# ------------------------------------------------------------------ #

_GRAPHML_NS = "http://graphml.graphdrawing.org/graphml"
_GRAPHML_KEY_DEFS = [
    ("name",             "node", "string"),
    ("source_urls",      "node", "string"),
    ("business_category","node", "string"),
    ("address",          "node", "string"),
    ("website",          "node", "string"),
    ("capital",          "node", "string"),
    ("employees",        "node", "string"),
    ("match_score",      "node", "double"),
    ("needs_review",     "node", "boolean"),
    ("relation_type",    "edge", "string"),
    ("edge_match_score", "edge", "double"),
    ("edge_needs_review","edge", "boolean"),
    ("target_raw_name",  "edge", "string"),
]


def write_graphml(
    nodes: dict[str, CompanyNode],
    edges: list[RelationEdge],
    output_path: Path,
) -> None:
    ET.register_namespace("", _GRAPHML_NS)
    root = ET.Element(f"{{{_GRAPHML_NS}}}graphml")

    # キー定義
    for kid, (attr, for_, atype) in enumerate(_GRAPHML_KEY_DEFS):
        k = ET.SubElement(root, f"{{{_GRAPHML_NS}}}key")
        k.set("id", f"d{kid}")
        k.set("for", for_)
        k.set("attr.name", attr)
        k.set("attr.type", atype)

    key_map = {attr: f"d{i}" for i, (attr, _, _) in enumerate(_GRAPHML_KEY_DEFS)}

    graph_el = ET.SubElement(root, f"{{{_GRAPHML_NS}}}graph")
    graph_el.set("id", "G")
    graph_el.set("edgedefault", "directed")

    def _data(parent: ET.Element, key: str, value: str | float | bool | None) -> None:
        if value is None:
            return
        d = ET.SubElement(parent, f"{{{_GRAPHML_NS}}}data")
        d.set("key", key_map[key])
        if isinstance(value, bool):
            d.text = "true" if value else "false"
        else:
            d.text = str(value)

    for node in nodes.values():
        n = ET.SubElement(graph_el, f"{{{_GRAPHML_NS}}}node")
        n.set("id", node.corporate_number)
        _data(n, "name", node.official_name)
        _data(n, "source_urls", "|".join(node.source_urls))
        _data(n, "business_category", node.business_category)
        _data(n, "address", node.address)
        _data(n, "website", node.website)
        _data(n, "capital", node.capital)
        _data(n, "employees", node.employees)
        _data(n, "match_score", node.match_score)
        _data(n, "needs_review", node.needs_review)

    for i, edge in enumerate(edges):
        e = ET.SubElement(graph_el, f"{{{_GRAPHML_NS}}}edge")
        e.set("id", f"e{i}")
        e.set("source", edge.source_corporate_number)
        e.set("target", edge.target_corporate_number)
        _data(e, "relation_type", edge.relation_type)
        _data(e, "edge_match_score", edge.match_score)
        _data(e, "edge_needs_review", edge.needs_review)
        _data(e, "target_raw_name", edge.target_raw_name)

    tree = ET.ElementTree(root)
    ET.indent(tree, space="  ")
    tree.write(str(output_path), encoding="utf-8", xml_declaration=True)
    logger.info("GraphML written: %s (%d nodes, %d edges)", output_path, len(nodes), len(edges))
