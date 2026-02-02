import logging
import math
import os
import threading
import time
from typing import List, Tuple

from crewai import Agent, Task, Crew, Process
from duckduckgo_search import DDGS
from duckduckgo_search.exceptions import RatelimitException
from fastapi import FastAPI, HTTPException
import openai as openai_module
from openai import OpenAI
from pydantic import BaseModel, Field


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI()

CACHE_TTL_SECONDS = int(os.getenv("RAG_SEARCH_CACHE_TTL_SECONDS", "86400"))
USE_DEEP_RESEARCH = os.getenv("RAG_USE_DEEP_RESEARCH", "true").lower() == "true"
ALLOW_DDG_FALLBACK = os.getenv("RAG_ALLOW_DUCKDUCKGO_FALLBACK", "false").lower() == "true"
STRICT_DEEP_RESEARCH = os.getenv("RAG_DEEP_RESEARCH_STRICT", "false").lower() == "true"
_cache_lock = threading.Lock()
_context_cache = {}


@app.on_event("startup")
def log_openai_version() -> None:
    version = getattr(openai_module, "__version__", "unknown")
    has_responses = hasattr(OpenAI(api_key=""), "responses")
    logger.info("openai version=%s responses_api=%s", version, has_responses)


class ReviewRequest(BaseModel):
    resume_text: str = Field(min_length=1)
    company_name: str = Field(min_length=1)
    job_title: str = Field(default="")


class ReviewResponse(BaseModel):
    report: str


def run_search(query: str, limit: int = 5) -> Tuple[List[dict], bool]:
    try:
        with DDGS() as ddgs:
            return list(ddgs.text(query, max_results=limit)), False
    except RatelimitException as exc:
        logger.warning("duckduckgo rate limited for query=%s error=%s", query, exc)
        return [], True
    except Exception as exc:
        logger.warning("duckduckgo search failed for query=%s error=%s", query, exc)
        return [], False


def get_cached_context(cache_key: str) -> List[str]:
    now = time.time()
    with _cache_lock:
        entry = _context_cache.get(cache_key)
        if not entry:
            return []
        timestamp, docs = entry
        if now-timestamp > CACHE_TTL_SECONDS:
            _context_cache.pop(cache_key, None)
            return []
        return docs


def set_cached_context(cache_key: str, docs: List[str]) -> None:
    if not docs:
        return
    with _cache_lock:
        _context_cache[cache_key] = (time.time(), docs)


def build_context(results: List[dict]) -> List[str]:
    docs = []
    for item in results:
        title = item.get("title", "")
        snippet = item.get("body", "")
        url = item.get("href", "")
        text = "Title: {title}\nSnippet: {snippet}\nURL: {url}".format(
            title=title,
            snippet=snippet,
            url=url,
        )
        docs.append(text.strip())
    return docs


def embed_texts(texts: List[str]) -> List[List[float]]:
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        raise HTTPException(status_code=500, detail="OPENAI_API_KEY is required")

    embedding_model = os.getenv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small")
    client = OpenAI(api_key=api_key)
    response = client.embeddings.create(model=embedding_model, input=texts)
    return [item.embedding for item in response.data]


def extract_output_text(response) -> str:
    output_text = getattr(response, "output_text", None)
    if output_text:
        return output_text.strip()
    choices = getattr(response, "choices", None)
    if choices:
        message = getattr(choices[0], "message", None)
        if message:
            content = getattr(message, "content", "")
            if content:
                return str(content).strip()
    outputs = getattr(response, "output", None)
    if not outputs:
        return ""
    parts = []
    for item in outputs:
        for content in getattr(item, "content", []):
            if getattr(content, "type", "") == "output_text":
                text = getattr(content, "text", "")
                if text:
                    parts.append(text.strip())
    return "\n".join(parts).strip()


def run_deep_research(company_name: str, job_title: str) -> str:
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        raise HTTPException(status_code=500, detail="OPENAI_API_KEY is required")
    model = os.getenv("OPENAI_DEEP_RESEARCH_MODEL", "o3-deep-research")
    fallback_model = os.getenv("OPENAI_DEEP_RESEARCH_FALLBACK_MODEL", "").strip()
    client = OpenAI(api_key=api_key)
    if not hasattr(client, "responses"):
        raise HTTPException(
            status_code=500,
            detail="Deep Research requires OpenAI responses API. Upgrade openai>=1.66 and rebuild the image.",
        )
    role = job_title or "指定なし"
    logger.info("deep research start model=%s company=%s role=%s", model, company_name, role)
    prompt = (
        "以下の企業について、採用に関わる価値観・求める人物像・評価軸・事業の特徴を、"
        "一次情報または信頼できる情報に基づいて簡潔に整理してください。"
        "誤りや不確実な点は断定せずに注意書きを入れてください。\n\n"
        "企業名: {company}\n"
        "職種: {role}\n"
        "出力は日本語で、箇条書きを含む短いレポート形式にしてください。"
    ).format(company=company_name, role=role)
    last_err = None
    def request_response(use_tools: bool, model_name: str):
        kwargs = {
            "model": model_name,
            "input": prompt,
            "temperature": 0.2,
            "max_output_tokens": 800,
        }
        if use_tools:
            kwargs["tools"] = [{"type": "web_search"}]
        return client.responses.create(**kwargs)

    for attempt in range(1, 3):
        try:
            response = request_response(True, model)
            output = extract_output_text(response)
            logger.info("deep research finished chars=%d attempt=%d", len(output), attempt)
            if output:
                return output
            logger.warning("deep research returned empty result attempt=%d", attempt)
        except Exception as exc:
            last_err = exc
            logger.warning("deep research failed attempt=%d error=%s", attempt, exc)
            if attempt == 1:
                fallback_name = fallback_model or model
                try:
                    response = request_response(False, fallback_name)
                    output = extract_output_text(response)
                    logger.info(
                        "deep research fallback finished chars=%d model=%s",
                        len(output),
                        fallback_name,
                    )
                    if output:
                        return output
                    logger.warning("deep research fallback returned empty result model=%s", fallback_name)
                except Exception as fallback_exc:
                    last_err = fallback_exc
                    logger.warning(
                        "deep research fallback failed model=%s error=%s",
                        fallback_name,
                        fallback_exc,
                    )
    raise last_err
    logger.info("deep research finished chars=%d", len(output))
    return output


def cosine_similarity(a: List[float], b: List[float]) -> float:
    dot = 0.0
    norm_a = 0.0
    norm_b = 0.0
    for av, bv in zip(a, b):
        dot += av * bv
        norm_a += av * av
        norm_b += bv * bv
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return dot / (math.sqrt(norm_a) * math.sqrt(norm_b))


def retrieve_docs(docs: List[str], query: str) -> List[str]:
    if not docs:
        return []
    embeddings = embed_texts(docs + [query])
    doc_embeddings = embeddings[:-1]
    query_embedding = embeddings[-1]

    scored = []
    for doc, emb in zip(docs, doc_embeddings):
        scored.append((cosine_similarity(query_embedding, emb), doc))
    scored.sort(key=lambda item: item[0], reverse=True)
    top_docs = [doc for _, doc in scored[: min(5, len(scored))]]
    return top_docs


def run_crewai(resume_text: str, company_name: str, job_title: str, context_docs: List[str]) -> str:
    context_block = "\n\n".join(context_docs)

    researcher = Agent(
        role="Company Researcher",
        goal="Extract company hiring signals and values from search results",
        backstory="You summarize key hiring signals for job applicants.",
        verbose=False,
    )

    reviewer = Agent(
        role="Resume Reviewer",
        goal="Produce a company-specific resume review report in Japanese",
        backstory="You are a professional career advisor.",
        verbose=False,
    )

    task_research = Task(
        description=(
            "Use the context to extract the company's core hiring signals. "
            "Return concise bullet keywords only.\n\n"
            "Company: {company}\n"
            "Role: {role}\n"
            "Context:\n{context}\n"
        ).format(company=company_name, role=job_title, context=context_block),
        expected_output="Bullet keywords",
        agent=researcher,
    )

    task_review = Task(
        description=(
            "Write the final report in Japanese, following this format:\n"
            "【企業別レビュー報告書】\n"
            "---\n"
            "#### ■ 対象企業\n"
            "{company}\n\n"
            "#### ■ この企業が求めている核心的要素\n"
            "- ...\n\n"
            "#### ■ 履歴書の最適化アドバイス\n"
            "- **強みの再定義**: ...\n"
            "- **不足している情報の補足**: ...\n\n"
            "#### ■ 修正後の自己PRイメージ\n"
            "...\n\n"
            "Use the resume text below and the extracted keywords. "
            "Keep it concise and practical.\n\n"
            "Company: {company}\n"
            "Role: {role}\n"
            "Resume:\n{resume}\n"
        ).format(company=company_name, role=job_title, resume=resume_text),
        expected_output="Final Japanese report in the requested format",
        agent=reviewer,
        context=[task_research],
    )

    crew = Crew(
        agents=[researcher, reviewer],
        tasks=[task_research, task_review],
        process=Process.sequential,
        verbose=False,
    )

    return str(crew.kickoff())


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.post("/resume/review", response_model=ReviewResponse)
def review_resume(request: ReviewRequest) -> ReviewResponse:
    role_label = request.job_title or "指定なし"

    cache_key = "{company}::{role}".format(company=request.company_name, role=role_label)
    retrieved = get_cached_context(cache_key)
    if retrieved:
        logger.info("duckduckgo cache hit for key=%s", cache_key)
    else:
        if USE_DEEP_RESEARCH and request.company_name.strip():
            try:
                report = run_deep_research(request.company_name, role_label)
                if report:
                    retrieved = [report]
                    set_cached_context(cache_key, retrieved)
                else:
                    logger.warning("deep research returned empty result")
            except Exception as exc:
                logger.warning("deep research failed error=%s", exc)
                if STRICT_DEEP_RESEARCH and not ALLOW_DDG_FALLBACK:
                    raise HTTPException(status_code=502, detail="Deep Research failed")

        if not retrieved and ALLOW_DDG_FALLBACK:
            logger.info("duckduckgo search start company=%s role=%s", request.company_name, role_label)
            query = "{company} {role} 求める人物像 大切にしている価値観".format(
                company=request.company_name,
                role=role_label,
            )
            results, rate_limited = run_search(query, limit=8)
            if not results:
                if rate_limited:
                    logger.warning("duckduckgo rate limited; continuing without external context")
                else:
                    logger.warning("duckduckgo returned no results; continuing without external context")
                retrieved = []
            else:
                docs = build_context(results)
                retrieved = retrieve_docs(docs, "採用 価値観 求める人物像")
                set_cached_context(cache_key, retrieved)
    report = run_crewai(
        resume_text=request.resume_text,
        company_name=request.company_name,
        job_title=role_label,
        context_docs=retrieved,
    )

    return ReviewResponse(report=report)
