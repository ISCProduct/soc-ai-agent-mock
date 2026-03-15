"""
gBizINFO API クライアント + 法人番号名寄せエンジン

マッチング戦略:
  優先①: 社名 + 郵便番号 の完全一致
  優先②: 社名 + 代表者名 の完全一致（将来拡張）
  フォールバック: 表記揺れ正規化後の部分一致 (difflib)

API ドキュメント: https://info.gbiz.go.jp/hojin/swagger-ui.html
認証: X-hojinInfo-api-token ヘッダー（環境変数 GBIZINFO_API_KEY）
"""
from __future__ import annotations

import logging
import os
import re
import time
import unicodedata
from difflib import SequenceMatcher
from functools import lru_cache

import requests

from models import CompanyNode, RawCompany

logger = logging.getLogger(__name__)

GBIZINFO_BASE = "https://info.gbiz.go.jp/hojin/v1/hojin"

# 法人格の正規化テーブル
_CORP_FORMS = {
    "（株）": "株式会社",
    "(株)": "株式会社",
    "㈱": "株式会社",
    "（有）": "有限会社",
    "(有)": "有限会社",
    "㈲": "有限会社",
    "（合）": "合同会社",
    "(合)": "合同会社",
    "（資）": "合資会社",
    "(資)": "合資会社",
    "（名）": "合名会社",
    "(名)": "合名会社",
}


def normalize_name(name: str) -> str:
    """社名の表記揺れを正規化する"""
    # Unicode NFKC 正規化（全角→半角 etc.）
    name = unicodedata.normalize("NFKC", name)
    # 略称展開
    for abbr, full in _CORP_FORMS.items():
        name = name.replace(abbr, full)
    # 余分な空白を除去
    name = re.sub(r"\s+", "", name)
    return name.strip()


def _similarity(a: str, b: str) -> float:
    return SequenceMatcher(None, a, b).ratio()


class GBizInfoClient:
    def __init__(self, api_key: str | None = None, rate_limit_sec: float = 1.0) -> None:
        self._api_key = api_key or os.environ.get("GBIZINFO_API_KEY", "")
        self._rate_limit = rate_limit_sec
        self._last_req: float = 0.0
        self._session = requests.Session()
        if self._api_key:
            self._session.headers["X-hojinInfo-api-token"] = self._api_key

    def _wait(self) -> None:
        elapsed = time.monotonic() - self._last_req
        if elapsed < self._rate_limit:
            time.sleep(self._rate_limit - elapsed)
        self._last_req = time.monotonic()

    @lru_cache(maxsize=512)
    def search(self, name: str, postal_code: str = "") -> list[dict]:
        """gBizINFO API を呼び出して候補を返す"""
        self._wait()
        params: dict = {"name": name, "limit": 10}
        if postal_code:
            params["postal_code"] = postal_code.replace("-", "")
        try:
            resp = self._session.get(GBIZINFO_BASE, params=params, timeout=10)
            if resp.status_code == 401:
                logger.warning("gBizINFO: API key is missing or invalid.")
                return []
            resp.raise_for_status()
            return resp.json().get("hojin-infos", [])
        except requests.RequestException as exc:
            logger.error("gBizINFO API error (%s): %s", name, exc)
            return []


class NameMatcher:
    """RawCompany → CompanyNode への名寄せを担う"""

    def __init__(self, client: GBizInfoClient, threshold: float = 0.75) -> None:
        self._client = client
        self._threshold = threshold

    def match(self, raw: RawCompany) -> CompanyNode | None:
        normalized = normalize_name(raw.raw_name)

        # ① 社名 + 郵便番号 で検索
        candidates = self._client.search(normalized, raw.postal_code)
        if not candidates and raw.postal_code:
            # 郵便番号なしで再検索
            candidates = self._client.search(normalized)

        if not candidates:
            logger.warning("gBizINFO: no candidates for '%s'", raw.raw_name)
            return self._fallback(raw)

        best, best_score = self._rank(normalized, candidates)
        needs_review = best_score < self._threshold

        if needs_review:
            logger.warning(
                "Low match score %.2f for '%s' → '%s'",
                best_score, raw.raw_name, best.get("name", ""),
            )

        return CompanyNode(
            corporate_number=best.get("corporate_number", ""),
            official_name=best.get("name", raw.raw_name),
            source_urls=[raw.source_url],
            business_category=best.get("business_summary", {}).get(
                "major_classification_name", ""
            ),
            address=best.get("location", raw.address),
            website=raw.website,
            capital=raw.capital,
            employees=raw.employees,
            match_score=best_score,
            needs_review=needs_review,
            related_companies_text=raw.related_companies_text,
            business_partners_text=raw.business_partners_text,
        )

    def _rank(self, query: str, candidates: list[dict]) -> tuple[dict, float]:
        scored = []
        for c in candidates:
            candidate_name = normalize_name(c.get("name", ""))
            score = _similarity(query, candidate_name)
            scored.append((c, score))
        scored.sort(key=lambda x: x[1], reverse=True)
        return scored[0]

    def _fallback(self, raw: RawCompany) -> CompanyNode:
        """API で見つからなかった場合のフォールバックノード"""
        return CompanyNode(
            corporate_number="UNKNOWN_" + normalize_name(raw.raw_name),
            official_name=raw.raw_name,
            source_urls=[raw.source_url],
            address=raw.address,
            website=raw.website,
            capital=raw.capital,
            employees=raw.employees,
            match_score=0.0,
            needs_review=True,
            related_companies_text=raw.related_companies_text,
            business_partners_text=raw.business_partners_text,
        )
