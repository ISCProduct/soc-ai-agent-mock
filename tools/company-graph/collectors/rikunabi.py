"""リクナビ クローラー

検索ページ(/n/job_search/)から求人票URL一覧を取得し、
各求人票詳細ページ(h2: '{会社名}｜{業種}')から会社名を抽出する。
"""
from __future__ import annotations

import logging
import re
from urllib.parse import urljoin

from collectors.base import BaseCrawler
from models import RawCompany

logger = logging.getLogger(__name__)

SITE_KEY = "rikunabi"
_JOB_DESC_RE = re.compile(r"/n/job_descriptions/([a-z0-9]+)/")
_BASE = "https://job.rikunabi.com"


class RikunabiCrawler(BaseCrawler):
    def search(self, keyword: str = "", max_pages: int = 5) -> list[str]:
        """検索結果ページから求人票URLを収集する。"""
        search_url = self.cfg.get("search_url", f"{_BASE}/n/job_search/")
        seen: set[str] = set()
        urls: list[str] = []

        for page in range(1, max_pages + 1):
            params = f"?keyword={keyword}&page={page}" if keyword else f"?page={page}"
            soup = self.get(search_url + params)
            if soup is None:
                break

            new_found = 0
            for a in soup.find_all("a", href=True):
                href = str(a["href"])
                if _JOB_DESC_RE.search(href):
                    full_url = urljoin(_BASE, href.split("?")[0])
                    if not full_url.endswith("/"):
                        full_url += "/"
                    if full_url not in seen:
                        seen.add(full_url)
                        urls.append(full_url)
                        new_found += 1
            logger.debug("[rikunabi] page %d: %d new job URLs", page, new_found)
            if new_found == 0:
                break

        return urls

    def parse_detail(self, url: str) -> RawCompany | None:
        soup = self.get(url)
        if soup is None:
            return None

        # h2 は "{会社名}｜{業種}" 形式
        h2 = soup.find("h2")
        if not h2:
            return None
        h2_text = h2.get_text(strip=True)
        company_name = h2_text.split("｜")[0].strip()
        if not company_name:
            return None

        h1 = soup.find("h1")
        description = h1.get_text(strip=True) if h1 else ""

        return RawCompany(
            source_site=SITE_KEY,
            source_url=url,
            raw_name=company_name,
            address="",
            postal_code="",
            website="",
            capital="",
            employees="",
            related_companies_text="",
            business_partners_text="",
            business_description=description,
        )
