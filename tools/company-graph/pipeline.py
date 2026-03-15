"""
マルチソース新卒採用データ × gBizINFO 統合解析パイプライン

Usage:
  python pipeline.py --sites mynavi rikunabi --query "IT" --pages 3 --out output/

環境変数:
  GBIZINFO_API_KEY  gBizINFO API トークン（必須）
"""
from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path

import yaml

from collectors.mynavi import MynaviCrawler
from collectors.rikunabi import RikunabiCrawler
from collectors.career_tasu import CareerTasuCrawler
from models import CompanyNode, RawCompany, RelationEdge
from normalization.gbizinfo import GBizInfoClient, NameMatcher
from relation.extractor import RelationExtractor
from graph.generator import write_json, write_graphml

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S",
)
logger = logging.getLogger(__name__)

_CRAWLERS = {
    "mynavi": MynaviCrawler,
    "rikunabi": RikunabiCrawler,
    "career_tasu": CareerTasuCrawler,
}

_CONFIG_PATH = Path(__file__).parent / "config" / "sites.yaml"


def load_config() -> dict:
    with _CONFIG_PATH.open(encoding="utf-8") as f:
        return yaml.safe_load(f)


def collect(sites: list[str], query: str, max_pages: int, config: dict) -> list[RawCompany]:
    """各媒体からRawCompanyを収集する"""
    all_companies: list[RawCompany] = []
    for site_name in sites:
        cls = _CRAWLERS.get(site_name)
        if cls is None:
            logger.warning("Unknown site: %s (skip)", site_name)
            continue
        site_cfg = config.get("sites", {}).get(site_name, {})
        crawler = cls(site_cfg)
        logger.info("Collecting from %s (query=%r, max_pages=%d)…", site_name, query, max_pages)
        try:
            companies = crawler.search(query, max_pages=max_pages)
            logger.info("  → %d companies fetched", len(companies))
            all_companies.extend(companies)
        except Exception as exc:
            logger.error("Failed to collect from %s: %s", site_name, exc)
    return all_companies


def normalize(
    raw_companies: list[RawCompany],
    matcher: NameMatcher,
) -> dict[str, CompanyNode]:
    """RawCompany → CompanyNode (法人番号をキーに名寄せ)"""
    nodes: dict[str, CompanyNode] = {}
    for raw in raw_companies:
        node = matcher.match(raw)
        if node is None:
            continue
        corp_num = node.corporate_number
        if corp_num in nodes:
            # 既存ノードに source_url をマージ
            existing = nodes[corp_num]
            merged_urls = list(dict.fromkeys(existing.source_urls + node.source_urls))
            nodes[corp_num] = CompanyNode(
                corporate_number=existing.corporate_number,
                official_name=existing.official_name,
                source_urls=merged_urls,
                business_category=existing.business_category or node.business_category,
                address=existing.address or node.address,
                website=existing.website or node.website,
                capital=existing.capital or node.capital,
                employees=existing.employees or node.employees,
                match_score=max(existing.match_score, node.match_score),
                needs_review=existing.needs_review or node.needs_review,
                related_companies_text=existing.related_companies_text or node.related_companies_text,
                business_partners_text=existing.business_partners_text or node.business_partners_text,
            )
        else:
            nodes[corp_num] = node
    logger.info("Normalized: %d unique companies", len(nodes))
    return nodes


def extract_edges(
    nodes: dict[str, CompanyNode],
    extractor: RelationExtractor,
) -> list[RelationEdge]:
    """全ノードのテキストから企業関係エッジを抽出する"""
    edges: list[RelationEdge] = []
    for node in nodes.values():
        edges.extend(extractor.extract(node, known_nodes=nodes))
    # 重複排除（同一 source-target-relation の組）
    seen: set[tuple[str, str, str]] = set()
    unique: list[RelationEdge] = []
    for edge in edges:
        key = (edge.source_corporate_number, edge.target_corporate_number, edge.relation_type)
        if key not in seen:
            seen.add(key)
            unique.append(edge)
    logger.info("Edges extracted: %d (after dedup)", len(unique))
    return unique


def run(
    sites: list[str],
    query: str,
    max_pages: int,
    output_dir: Path,
    gbizinfo_api_key: str | None = None,
    match_threshold: float = 0.75,
) -> None:
    config = load_config()
    output_dir.mkdir(parents=True, exist_ok=True)

    # Phase 1: スクレイピング
    raw_companies = collect(sites, query, max_pages, config)
    if not raw_companies:
        logger.error("No companies collected. Exiting.")
        sys.exit(1)

    # Phase 2: 名寄せ
    client = GBizInfoClient(api_key=gbizinfo_api_key)
    matcher = NameMatcher(client, threshold=match_threshold)
    nodes = normalize(raw_companies, matcher)

    # Phase 3: 関係抽出 + グラフ出力
    extractor = RelationExtractor(matcher=matcher, client=client)
    edges = extract_edges(nodes, extractor)

    json_path = output_dir / "company_graph.json"
    graphml_path = output_dir / "company_graph.graphml"
    write_json(nodes, edges, json_path)
    write_graphml(nodes, edges, graphml_path)

    needs_review_nodes = sum(1 for n in nodes.values() if n.needs_review)
    needs_review_edges = sum(1 for e in edges if e.needs_review)
    logger.info(
        "Done. nodes=%d (review=%d), edges=%d (review=%d)",
        len(nodes), needs_review_nodes,
        len(edges), needs_review_edges,
    )
    logger.info("Output: %s, %s", json_path, graphml_path)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="マルチソース新卒採用データ × gBizINFO 統合解析パイプライン"
    )
    parser.add_argument(
        "--sites",
        nargs="+",
        default=["mynavi", "rikunabi", "career_tasu"],
        choices=list(_CRAWLERS.keys()),
        help="対象サイト (default: all)",
    )
    parser.add_argument(
        "--query",
        default="IT",
        help="検索キーワード (default: IT)",
    )
    parser.add_argument(
        "--pages",
        type=int,
        default=3,
        metavar="N",
        help="各サイトの最大取得ページ数 (default: 3)",
    )
    parser.add_argument(
        "--out",
        type=Path,
        default=Path("output"),
        help="出力ディレクトリ (default: output/)",
    )
    parser.add_argument(
        "--api-key",
        default=None,
        help="gBizINFO API キー (未指定時は環境変数 GBIZINFO_API_KEY を使用)",
    )
    parser.add_argument(
        "--threshold",
        type=float,
        default=0.75,
        help="名寄せ一致スコアの閾値 0–1 (default: 0.75)",
    )
    args = parser.parse_args()

    run(
        sites=args.sites,
        query=args.query,
        max_pages=args.pages,
        output_dir=args.out,
        gbizinfo_api_key=args.api_key,
        match_threshold=args.threshold,
    )


if __name__ == "__main__":
    main()
