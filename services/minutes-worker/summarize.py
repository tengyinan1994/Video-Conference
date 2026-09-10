"""LangChain + OpenAI 兼容（DeepSeek 等）生成结构化纪要。"""

from __future__ import annotations

import json
import logging
import os
from typing import Any

from langchain_core.prompts import ChatPromptTemplate
from langchain_openai import ChatOpenAI
from pydantic import BaseModel, Field

log = logging.getLogger("minutes-worker.summarize")


class MinutesStructured(BaseModel):
    summary: str = Field(description="会议纪要正文，中文，分点清晰")
    todos: list[str] = Field(default_factory=list, description="待办事项")
    decisions: list[str] = Field(default_factory=list, description="决议")
    risks: list[str] = Field(default_factory=list, description="风险或需跟进点")


def summarize_meeting(*, title: str, attendees: list[str], transcript: str) -> dict[str, Any]:
    base_url = os.getenv("OPENAI_BASE_URL", "https://api.deepseek.com/v1")
    api_key = os.getenv("OPENAI_API_KEY", "")
    model = os.getenv("OPENAI_MODEL", "deepseek-chat")
    if not api_key or os.getenv("MOCK_LLM", "").lower() in ("1", "true", "yes"):
        log.warning("using mock LLM summary (no OPENAI_API_KEY or MOCK_LLM=1)")
        return {
            "summary": f"【mock 纪要】《{title or '未命名会议'}》\n参会：{'、'.join(attendees) or '未知'}\n\n根据转写整理（测试占位）：\n" + transcript[:800],
            "todos": ["（mock）请配置 OPENAI_API_KEY 后重新生成真实纪要"],
            "decisions": [],
            "risks": [],
            "model": "mock",
        }

    llm = ChatOpenAI(
        base_url=base_url,
        api_key=api_key,
        model=model,
        temperature=0.2,
    )
    structured = llm.with_structured_output(MinutesStructured)
    prompt = ChatPromptTemplate.from_messages(
        [
            (
                "system",
                "你是企业会议纪要助手。根据转写生成简洁中文纪要与结构化字段。"
                "不要编造转写中未出现的事实；待办要可执行。",
            ),
            (
                "human",
                "会议标题：{title}\n参会人：{attendees}\n\n转写全文：\n{transcript}",
            ),
        ]
    )
    chain = prompt | structured
    try:
        out: MinutesStructured = chain.invoke(
            {
                "title": title or "未命名会议",
                "attendees": "、".join(attendees) if attendees else "未知",
                "transcript": transcript,
            }
        )
    except Exception as e:
        log.warning("structured_output failed, fallback json: %s", e)
        return summarize_meeting_fallback_json(title=title, attendees=attendees, transcript=transcript)
    log.info("summarize done model=%s todos=%d", model, len(out.todos))
    return {
        "summary": out.summary,
        "todos": out.todos,
        "decisions": out.decisions,
        "risks": out.risks,
        "model": model,
    }


def summarize_meeting_fallback_json(*, title: str, attendees: list[str], transcript: str) -> dict[str, Any]:
    """无 structured_output 时的兜底（一般不用）。"""
    base_url = os.getenv("OPENAI_BASE_URL", "https://api.deepseek.com/v1")
    api_key = os.getenv("OPENAI_API_KEY", "")
    model = os.getenv("OPENAI_MODEL", "deepseek-chat")
    llm = ChatOpenAI(base_url=base_url, api_key=api_key, model=model, temperature=0.2)
    prompt = (
        f"会议：{title}\n参会：{', '.join(attendees)}\n转写：\n{transcript}\n"
        '请输出 JSON：{"summary":"...","todos":[],"decisions":[],"risks":[]}'
    )
    msg = llm.invoke(prompt)
    text = getattr(msg, "content", "") or ""
    start = text.find("{")
    end = text.rfind("}")
    if start >= 0 and end > start:
        data = json.loads(text[start : end + 1])
        data["model"] = model
        return data
    return {"summary": text, "todos": [], "decisions": [], "risks": [], "model": model}
