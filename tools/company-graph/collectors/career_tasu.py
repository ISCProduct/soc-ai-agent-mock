"""キャリタス就活 クローラー

検索: /condition-search/result/?keyword={kw}&p={page}
     → /corp/{id}/default/ へのリンクを収集
詳細: /corp/{id}/default/ からページテキストで会社情報を取得
     （SSR済みのためBeautifulSoupで取得可能）
"""
from __future__ import annotations

import logging
import re
from urllib.parse import urljoin

from collectors.base import BaseCrawler
from models import RawCompany

logger = logging.getLogger(__name__)

SITE_KEY = "career_tasu"
_BASE = "https://job.career-tasu.jp"
_CORP_PATH_RE = re.compile(r"/corp/\d+/")


class CareerTasuCrawler(BaseCrawler):
    def search(self, keyword: str = "", max_pages: int = 5) -> list[str]:
        """検索結果ページから企業詳細URLを収集する。"""
        search_url = self.cfg.get("search_url", f"{_BASE}/condition-search/result/")
        seen: set[str] = set()
        urls: list[str] = []

        for page in range(1, max_pages + 1):
            params = f"?keyword={keyword}&p={page}" if keyword else f"?p={page}"
            soup = self.get(search_url + params)
            if soup is None:
                break

            new_found = 0
            for a in soup.find_all("a", href=True):
                href = str(a["href"])
                if _CORP_PATH_RE.search(href):
                    full_url = urljoin(_BASE, href.split("?")[0])
                    if not full_url.endswith("/"):
                        full_url += "/"
                    if full_url not in seen:
                        seen.add(full_url)
                        urls.append(full_url)
                        new_found += 1
            logger.debug("[career_tasu] page %d: %d new corp URLs", page, new_found)
            if new_found == 0:
                break

        return urls

    def parse_detail(self, url: str) -> RawCompany | None:
        soup = self.get(url)
        if soup is None:
            return None

        # ページテキスト全体から会社情報を正規表現で抽出
        text = soup.get_text(separator="\n", strip=True)

        # 会社名: title タグ "株式会社XX | キャリタス就活" or h1
        company_name = ""
        title_el = soup.find("title")
        if title_el:
            title_text = title_el.get_text(strip=True)
            company_name = title_text.split("|")[0].split("の")[0].strip()
        if not company_name:
            h1 = soup.find("h1")
            if h1:
                company_name = h1.get_text(strip=True)
        if not company_name:
            return None

        # 資本金: "資本金： {value}" パターン
        capital = ""
        m = re.search(r"資本金[：:]\s*([^\n]{1,30})", text)
        if m:
            capital = m.group(1).strip()

        # 売上高/従業員数
        employees = ""
        m = re.search(r"従業員数[：:]\s*([^\n]{1,30})", text)
        if m:
            employees = m.group(1).strip()

        # 都道府県をアドレスとして取得（検索リンクのテキストに含まれることが多い）
        address = ""
        pref_m = re.search(
            r"(北海道|青森|岩手|宮城|秋田|山形|福島|茨城|栃木|群馬|埼玉|千葉|東京|神奈川"
            r"|新潟|富山|石川|福井|山梨|長野|岐阜|静岡|愛知|三重|滋賀|京都|大阪|兵庫|奈良"
            r"|和歌山|鳥取|島根|岡山|広島|山口|徳島|香川|愛媛|高知|福岡|佐賀|長崎|熊本"
            r"|大分|宮崎|鹿児島|沖縄)[都道府県]?", text
        )
        if pref_m:
            address = pref_m.group(0)

        return RawCompany(
            source_site=SITE_KEY,
            source_url=url,
            raw_name=company_name,
            address=address,
            postal_code=self._extract_postal(address),
            website="",
            capital=capital,
            employees=employees,
            related_companies_text="",
            business_partners_text="",
            business_description="",
        )
