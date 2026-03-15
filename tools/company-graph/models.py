"""共通データモデル定義"""
from __future__ import annotations
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class RawCompany:
    """スクレイピング直後の生データ"""
    source_site: str
    source_url: str
    raw_name: str
    address: str = ""
    postal_code: str = ""
    website: str = ""
    capital: str = ""
    employees: str = ""
    related_companies_text: str = ""
    business_partners_text: str = ""
    business_description: str = ""


@dataclass
class CompanyNode:
    """法人番号で正規化済みの企業ノード"""
    corporate_number: str                        # 法人番号（主キー）
    official_name: str                           # 法人格を含む正式名称
    source_urls: list[str] = field(default_factory=list)
    business_category: str = ""                  # gBizINFO 産業分類
    address: str = ""
    website: str = ""
    capital: str = ""
    employees: str = ""
    match_score: float = 1.0                     # 名寄せ信頼スコア (0-1)
    needs_review: bool = False                   # 目視確認フラグ
    # 関係抽出の元テキスト
    related_companies_text: str = ""
    business_partners_text: str = ""

    def merge(self, other: "CompanyNode") -> None:
        """同一法人番号の情報をマージする"""
        for url in other.source_urls:
            if url not in self.source_urls:
                self.source_urls.append(url)
        if not self.business_category and other.business_category:
            self.business_category = other.business_category
        if not self.website and other.website:
            self.website = other.website
        if other.related_companies_text:
            self.related_companies_text = (
                self.related_companies_text + "\n" + other.related_companies_text
            ).strip()
        if other.business_partners_text:
            self.business_partners_text = (
                self.business_partners_text + "\n" + other.business_partners_text
            ).strip()


@dataclass
class RelationEdge:
    """企業間の関係エッジ"""
    source_corporate_number: str
    target_corporate_number: str
    target_raw_name: str          # 名寄せ前の生社名（デバッグ用）
    relation_type: str            # "capital" | "partner" | "group"
    match_score: float = 1.0
    needs_review: bool = False
