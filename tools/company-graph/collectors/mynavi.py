"""マイナビ クローラー

マイナビの企業検索はログイン・セッション・JS が必要なため、
現状は search() で空リストを返し警告のみ記録する。

将来的に Playwright 等を導入した場合はここに実装を追加する。
"""
from __future__ import annotations

import logging

from collectors.base import BaseCrawler
from models import RawCompany

logger = logging.getLogger(__name__)

SITE_KEY = "mynavi"


class MynaviCrawler(BaseCrawler):
    def search(self, keyword: str = "", max_pages: int = 5) -> list[str]:
        logger.warning(
            "マイナビは企業検索にログイン・JS が必要なためスキップします。"
            " Playwright 対応後に利用可能になります。"
        )
        return []

    def parse_detail(self, url: str) -> RawCompany | None:
        return None
