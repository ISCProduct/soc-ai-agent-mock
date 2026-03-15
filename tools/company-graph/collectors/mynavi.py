"""マイナビ2027 クローラー"""
from __future__ import annotations

import logging
from urllib.parse import urljoin

from collectors.base import BaseCrawler
from models import RawCompany

logger = logging.getLogger(__name__)

SITE_KEY = "mynavi"


class MynaviCrawler(BaseCrawler):
    def search(self, keyword: str = "", max_pages: int = 5) -> list[str]:
        sel = self.selectors
        urls: list[str] = []
        search_url = self.cfg.get("search_url", self.base_url)

        for page in range(1, max_pages + 1):
            page_url = f"{search_url}?page={page}"
            if keyword:
                page_url += f"&keyword={keyword}"
            soup = self.get(page_url)
            if soup is None:
                break
            items = soup.select(sel.get("company_list", ""))
            if not items:
                break
            for item in items:
                a = item.select_one(sel.get("company_link", "a"))
                if a and a.get("href"):
                    urls.append(urljoin(self.base_url, a["href"]))
            logger.debug("[mynavi] page %d: %d items", page, len(items))
        return urls

    def parse_detail(self, url: str) -> RawCompany | None:
        soup = self.get(url)
        if soup is None:
            return None
        d = self.selectors.get("detail", {})
        address = self._text(soup, d.get("address", ""))
        return RawCompany(
            source_site=SITE_KEY,
            source_url=url,
            raw_name=self._text(soup, d.get("company_name", "h1")),
            address=address,
            postal_code=self._extract_postal(address),
            website=self._attr(soup, d.get("website", ""), "href"),
            capital=self._text(soup, d.get("capital", "")),
            employees=self._text(soup, d.get("employees", "")),
            related_companies_text=self._text(soup, d.get("related_companies", "")),
            business_partners_text=self._text(soup, d.get("business_partners", "")),
            business_description=self._text(soup, d.get("business_description", "")),
        )
