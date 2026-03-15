"""
基底クローラー
- robots.txt チェック
- レート制限
- User-Agent 設定
- config.yaml からセレクタを動的ロード
"""
from __future__ import annotations

import logging
import re
import time
from abc import ABC, abstractmethod
from urllib.parse import urljoin, urlparse
from urllib.robotparser import RobotFileParser

import requests
from bs4 import BeautifulSoup

from models import RawCompany

logger = logging.getLogger(__name__)


class BaseCrawler(ABC):
    def __init__(self, site_config: dict) -> None:
        self.cfg = site_config
        self.base_url: str = site_config["base_url"]
        self.rate_limit: float = float(site_config.get("rate_limit_sec", 3.0))
        self.user_agent: str = site_config.get(
            "user_agent",
            "Mozilla/5.0 (compatible; CompanyGraphBot/1.0)",
        )
        self.selectors: dict = site_config.get("selectors", {})
        self._session = requests.Session()
        self._session.headers.update({"User-Agent": self.user_agent})
        self._robots = self._load_robots()
        self._last_request: float = 0.0

    # ------------------------------------------------------------------ #
    # robots.txt
    # ------------------------------------------------------------------ #
    def _load_robots(self) -> RobotFileParser:
        rp = RobotFileParser()
        robots_url = urljoin(self.base_url, "/robots.txt")
        try:
            rp.set_url(robots_url)
            rp.read()
            logger.info("robots.txt loaded: %s", robots_url)
        except Exception as exc:
            logger.warning("robots.txt load failed (%s): %s", robots_url, exc)
        return rp

    def can_fetch(self, url: str) -> bool:
        allowed = self._robots.can_fetch(self.user_agent, url)
        if not allowed:
            logger.warning("robots.txt disallows: %s", url)
        return allowed

    # ------------------------------------------------------------------ #
    # HTTP
    # ------------------------------------------------------------------ #
    def _wait(self) -> None:
        elapsed = time.monotonic() - self._last_request
        if elapsed < self.rate_limit:
            time.sleep(self.rate_limit - elapsed)
        self._last_request = time.monotonic()

    def get(self, url: str, **kwargs) -> BeautifulSoup | None:
        if not self.can_fetch(url):
            return None
        self._wait()
        try:
            resp = self._session.get(url, timeout=15, **kwargs)
            resp.raise_for_status()
            resp.encoding = resp.apparent_encoding
            return BeautifulSoup(resp.text, "html.parser")
        except requests.RequestException as exc:
            logger.error("GET %s failed: %s", url, exc)
            return None

    # ------------------------------------------------------------------ #
    # セレクタユーティリティ
    # ------------------------------------------------------------------ #
    @staticmethod
    def _text(soup: BeautifulSoup, selector: str) -> str:
        el = soup.select_one(selector)
        return el.get_text(strip=True) if el else ""

    @staticmethod
    def _attr(soup: BeautifulSoup, selector: str, attr: str) -> str:
        el = soup.select_one(selector)
        return el.get(attr, "") if el else ""

    @staticmethod
    def _extract_postal(address: str) -> str:
        m = re.search(r"〒?\s*(\d{3}[-ー]\d{4})", address)
        return m.group(1).replace("ー", "-") if m else ""

    # ------------------------------------------------------------------ #
    # 公開インターフェース
    # ------------------------------------------------------------------ #
    @abstractmethod
    def search(self, keyword: str = "", max_pages: int = 5) -> list[str]:
        """企業詳細ページの URL リストを返す"""

    @abstractmethod
    def parse_detail(self, url: str) -> RawCompany | None:
        """企業詳細ページを解析して RawCompany を返す"""

    def crawl(self, keyword: str = "", max_pages: int = 5) -> list[RawCompany]:
        """search → parse_detail を一気通貫で実行する"""
        urls = self.search(keyword=keyword, max_pages=max_pages)
        logger.info("[%s] %d companies to scrape", self.cfg["name"], len(urls))
        results: list[RawCompany] = []
        for url in urls:
            company = self.parse_detail(url)
            if company:
                results.append(company)
        return results
