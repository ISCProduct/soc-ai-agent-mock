"""
企業関係抽出エンジン

GiNZA（日本語 NLP）が利用可能な場合は固有表現認識（NER）で社名を抽出する。
利用不可の場合は正規表現ベースのフォールバックで抽出する。

出力する関係種別:
  capital  - 資本関係（親会社・子会社・持分法適用）
  group    - グループ会社・関連会社
  partner  - 主要取引先
"""
from __future__ import annotations

import logging
import re

from models import CompanyNode, RelationEdge
from normalization.gbizinfo import GBizInfoClient, NameMatcher, normalize_name

logger = logging.getLogger(__name__)

# GiNZA のオプショナルインポート
try:
    import spacy  # type: ignore
    _NLP = spacy.load("ja_ginza")
    logger.info("GiNZA NLP model loaded.")
except Exception:
    _NLP = None
    logger.info("GiNZA not available; using regex fallback for name extraction.")

# 関連会社テキストから社名を区切るパターン
_SPLIT_PATTERN = re.compile(r"[、，,・\n\r/／｜|]+")

# 法人格が含まれている文字列を社名候補とみなす（正規表現）
_CORP_KEYWORDS = re.compile(
    r"(?:株式会社|有限会社|合同会社|合資会社|合名会社|一般社団法人|公益社団法人"
    r"|一般財団法人|公益財団法人|学校法人|医療法人|社会福祉法人|協同組合"
    r"|（株）|\(株\)|㈱|（有）|\(有\)|㈲)"
)


def _extract_names_ginza(text: str) -> list[str]:
    """GiNZA NER で組織名を抽出する"""
    doc = _NLP(text)
    names = [ent.text for ent in doc.ents if ent.label_ in ("ORG", "PRODUCT")]
    return [n for n in names if _CORP_KEYWORDS.search(n)]


def _extract_names_regex(text: str) -> list[str]:
    """正規表現で法人格付き社名を抽出する（フォールバック）"""
    # 区切り文字で分割してから法人格キーワードを含むものを抽出
    parts = _SPLIT_PATTERN.split(text)
    names: list[str] = []
    for part in parts:
        part = part.strip()
        if part and _CORP_KEYWORDS.search(part):
            names.append(part)
    return names


def extract_company_names(text: str) -> list[str]:
    if not text:
        return []
    if _NLP is not None:
        names = _extract_names_ginza(text)
        if names:
            return names
    return _extract_names_regex(text)


class RelationExtractor:
    """CompanyNode のテキストフィールドから RelationEdge を生成する"""

    def __init__(self, matcher: NameMatcher, client: GBizInfoClient) -> None:
        self._matcher = matcher
        self._client = client

    def extract(
        self,
        node: CompanyNode,
        known_nodes: dict[str, CompanyNode],
    ) -> list[RelationEdge]:
        edges: list[RelationEdge] = []

        # 関連会社・グループ会社テキスト
        edges.extend(
            self._resolve(
                source=node,
                text=node.related_companies_text,
                relation_type="group",
                known_nodes=known_nodes,
            )
        )
        # 主要取引先テキスト
        edges.extend(
            self._resolve(
                source=node,
                text=node.business_partners_text,
                relation_type="partner",
                known_nodes=known_nodes,
            )
        )
        return edges

    def _resolve(
        self,
        source: CompanyNode,
        text: str,
        relation_type: str,
        known_nodes: dict[str, CompanyNode],
    ) -> list[RelationEdge]:
        if not text:
            return []
        edges: list[RelationEdge] = []
        raw_names = extract_company_names(text)
        for raw_name in raw_names:
            normalized = normalize_name(raw_name)
            # 既知ノードに完全一致があれば高速パス
            target_node = self._find_in_known(normalized, known_nodes)
            if target_node:
                edges.append(
                    RelationEdge(
                        source_corporate_number=source.corporate_number,
                        target_corporate_number=target_node.corporate_number,
                        target_raw_name=raw_name,
                        relation_type=relation_type,
                        match_score=1.0,
                    )
                )
            else:
                # gBizINFO で新たに検索
                candidates = self._client.search(normalized)
                if candidates:
                    best = candidates[0]
                    corp_num = best.get("corporate_number", "UNKNOWN_" + normalized)
                    edges.append(
                        RelationEdge(
                            source_corporate_number=source.corporate_number,
                            target_corporate_number=corp_num,
                            target_raw_name=raw_name,
                            relation_type=relation_type,
                            match_score=0.8,
                            needs_review=True,
                        )
                    )
                else:
                    edges.append(
                        RelationEdge(
                            source_corporate_number=source.corporate_number,
                            target_corporate_number="UNKNOWN_" + normalized,
                            target_raw_name=raw_name,
                            relation_type=relation_type,
                            match_score=0.0,
                            needs_review=True,
                        )
                    )
        return edges

    @staticmethod
    def _find_in_known(
        normalized: str, known_nodes: dict[str, CompanyNode]
    ) -> CompanyNode | None:
        for node in known_nodes.values():
            if normalize_name(node.official_name) == normalized:
                return node
        return None
