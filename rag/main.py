import math
import os
from typing import List

from crewai import Agent, Task, Crew, Process
from duckduckgo_search import DDGS
from fastapi import FastAPI, HTTPException
from openai import OpenAI
from pydantic import BaseModel, Field


app = FastAPI()


class ReviewRequest(BaseModel):
    resume_text: str = Field(min_length=1)
    company_name: str = Field(min_length=1)
    job_title: str = Field(default="")


class ReviewResponse(BaseModel):
    report: str


def run_search(query: str, limit: int = 5) -> List[dict]:
    with DDGS() as ddgs:
        return list(ddgs.text(query, max_results=limit))


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

    search_queries = [
        "{company} {role} 求める人物像".format(company=request.company_name, role=role_label),
        "{company} 中途採用 大切にしている価値観".format(company=request.company_name),
    ]

    results = []
    for query in search_queries:
        results.extend(run_search(query, limit=5))

    if not results:
        raise HTTPException(status_code=502, detail="No search results found")

    docs = build_context(results)
    retrieved = retrieve_docs(docs, "採用 価値観 求める人物像")
    report = run_crewai(
        resume_text=request.resume_text,
        company_name=request.company_name,
        job_title=role_label,
        context_docs=retrieved,
    )

    return ReviewResponse(report=report)
