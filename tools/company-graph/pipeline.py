"""
マルチソース新卒採用データ × gBizINFO 統合解析パイプライン

Usage:
  python pipeline.py --sites mynavi rikunabi --query "IT" --pages 3 --out output/
  python pipeline.py --year 2028  # 年度を手動指定（省略時は自動計算）

環境変数:
  GBIZINFO_API_KEY  gBizINFO API トークン（必須）

年度自動計算ルール:
  4月以降: 当年 + 2（例: 2026年4月 → 2028年度）
  3月以前: 当年 + 1（例: 2026年3月 → 2027年度）
"""
from __future__ import annotations

import argparse
import datetime
import json
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


def resolve_year(override: int | None = None) -> int:
    """対象卒業年度を返す。

    - override が指定された場合はそれを使用。
    - 4月以降: 当年 + 2（新年度の採用サイトが開設されるタイミング）
    - 3月以前: 当年 + 1（現行メインシーズン）
    """
    if override:
        return override
    today = datetime.date.today()
    return today.year + 2 if today.month >= 4 else today.year + 1


def apply_year_template(config: dict, year: int) -> dict:
    """設定値中の {year} / {year2d} プレースホルダーを展開する。"""
    year2d = str(year)[-2:]
    raw = json.dumps(config, ensure_ascii=False)
    raw = raw.replace("{year}", str(year)).replace("{year2d}", year2d)
    return json.loads(raw)


def load_config(year: int | None = None) -> tuple[dict, int]:
    """設定を読み込み、年度プレースホルダーを展開して返す。"""
    with _CONFIG_PATH.open(encoding="utf-8") as f:
        raw_config = yaml.safe_load(f)
    target_year = resolve_year(year)
    config = apply_year_template(raw_config, target_year)
    logger.info("Target graduation year: %d", target_year)
    return config, target_year


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
            companies = crawler.crawl(keyword=query, max_pages=max_pages)
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
    year: int | None = None,
) -> None:
    config, target_year = load_config(year)
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
    parser.add_argument(
        "--year",
        type=int,
        default=None,
        metavar="YYYY",
        help=(
            "対象卒業年度を手動指定 (例: 2028)。"
            "省略時は自動計算: 4月以降 → 当年+2、3月以前 → 当年+1"
        ),
    )
    args = parser.parse_args()

    run(
        sites=args.sites,
        query=args.query,
        max_pages=args.pages,
        output_dir=args.out,
        gbizinfo_api_key=args.api_key,
        match_threshold=args.threshold,
        year=args.year,
    )


if __name__ == "__main__":
    main()
