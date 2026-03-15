"""
company-graph マイクロサービス

FastAPI で pipeline.py をラップし、HTTP API として公開する。
"""
from __future__ import annotations

import logging
import os
import sys
from pathlib import Path

from fastapi import FastAPI
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

# pipeline モジュールを同一ディレクトリから import できるようにする
sys.path.insert(0, str(Path(__file__).parent))

from pipeline import run, resolve_year  # noqa: E402

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S",
)

app = FastAPI(title="company-graph API", version="1.0")

OUTPUT_DIR = Path(os.environ.get("OUTPUT_DIR", "/app/output"))


class CrawlRequest(BaseModel):
    sites: list[str] = Field(default=["mynavi", "rikunabi", "career_tasu"])
    query: str = Field(default="IT")
    pages: int = Field(default=2, ge=1, le=20)
    year: int | None = Field(default=None)
    threshold: float = Field(default=0.75, ge=0.0, le=1.0)


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.get("/target-year")
def target_year(year: int | None = None) -> dict:
    return {"target_year": resolve_year(year)}


@app.post("/crawl")
def crawl(req: CrawlRequest) -> JSONResponse:
    log_messages: list[str] = []

    # ログを収集するハンドラ
    class ListHandler(logging.Handler):
        def emit(self, record: logging.LogRecord) -> None:
            log_messages.append(self.format(record))

    handler = ListHandler()
    handler.setFormatter(logging.Formatter("%(asctime)s [%(levelname)s] %(name)s: %(message)s"))
    root_logger = logging.getLogger()
    root_logger.addHandler(handler)

    try:
        run(
            sites=req.sites,
            query=req.query,
            max_pages=req.pages,
            output_dir=OUTPUT_DIR,
            gbizinfo_api_key=os.environ.get("GBIZINFO_API_KEY"),
            match_threshold=req.threshold,
            year=req.year,
        )
        return JSONResponse(
            content={"ok": True, "logs": "\n".join(log_messages), "output_dir": str(OUTPUT_DIR)},
            status_code=200,
        )
    except SystemExit:
        return JSONResponse(
            content={"ok": False, "logs": "\n".join(log_messages), "error": "No companies collected"},
            status_code=422,
        )
    except Exception as exc:
        return JSONResponse(
            content={"ok": False, "logs": "\n".join(log_messages), "error": str(exc)},
            status_code=500,
        )
    finally:
        root_logger.removeHandler(handler)
