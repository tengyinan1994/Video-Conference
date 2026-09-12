"""LangChain + OpenAI 兼容（DeepSeek 等）生成中文会议纪要（Markdown）。"""

from __future__ import annotations

import json
import logging
import os
from typing import Any

from langchain_core.messages import HumanMessage, SystemMessage
from langchain_openai import ChatOpenAI

log = logging.getLogger("minutes-worker.summarize")

# 输出骨架：与 services/minutes-worker/minutes_template.md 保持一致，改动请两边同步。
MINUTES_SKELETON = """## 会议概览

- **会议主题**：{一句话}
- **会议摘要**：{2~4 句：这是一场什么会 → 主要讨论/排查了什么 → 结论或当前状态}
- **主要发言人**：{仅当转写能区分说话人时才输出该行，否则整行删除}

## 一、{议题一}

### 1. {子议题}

- **{要点标签}**：{具体内容}
- **{要点标签}**：{具体内容}

### 2. {子议题}

- **{要点标签}**：{具体内容}

## 二、{议题二}

### 1. {子议题}

- **{要点标签}**：{具体内容}

## 决议事项

- **{决议要点}**：{已拍板的结论}

## 待办事项

- [ ] {动词开头的动作}（负责人：{姓名}；时限：{时间}）

## 风险与遗留问题

- **{风险名}**：{会造成什么影响}"""

MINUTES_RULES = """写作规则（必须全部遵守）：

【结构与粒度】
1. 只输出骨架里的章节，不得增删章节标题、不得改变层级；正文从「## 会议概览」开始，不要重复输出会议标题。
2. 议题章节 2~6 个，每个议题下子议题 1~4 个，每个子议题下要点 1~5 条。
3. 议题名用动宾短语（如「视频会议音频回音排查」），禁止「其他」「补充说明」这类空标题。
4. 每条要点写成「**标签**：内容」，标签 4~10 字，内容不超过 60 字，一条只说一件事。
5. 要点必须落到具体信息（数字、现象、结论、决定），禁止「进行了深入讨论」「大家积极发言」这类空话。
6. 按信息重要度排序，不按发言时间写流水账；同一议题内的要点合并去重。

【事实约束】
7. 只写转写里真实出现的内容；没提到或听不清的一律不写，宁可少一条，不许猜测补全。
8. 人名、数字、版本号、机型等照抄转写，不做「大概」「似乎」式推断。
9. 无法确认的归因不要落到某个人头上（如「某成员未戴耳机」），改写为现象描述（如「部分人员佩戴耳机后仍存在回音」）。

【三节定性（去重的关键）】
10. 决议事项：明确拍板的结论，有「确定 / 决定 / 就用这个方案 / 不再做 X」的语义；仅陈述现状的不算决议。
11. 待办事项：动词开头的可执行动作，一条只做一件事；只有转写里提到责任人或时限时才补括号。
12. 风险与遗留问题：本轮未解决、且尚无明确下一步动作的隐患，只写影响与不确定性。
13. 同一事实只能出现在决议 / 待办 / 风险中的一节，优先级：待办事项 > 决议事项 > 风险与遗留问题。判断方法：某个隐患一旦已经拆成具体动作，就只留在待办里，风险一节不得再写它。例：已列「排期 50 人并发测试」到待办，风险一节就不要再写「50 人规模未验证」。
14. 决议事项 / 待办事项 / 风险与遗留问题三节没有内容就整节删除，禁止写「无」「暂无」或留空占位。

【排版】
15. 全篇中文、使用全角标点；数字与英文术语保持原样（如 FunASR、50 人）。
16. 正文总长 600~1500 字，待办不超过 8 条。
17. 只输出 Markdown 正文，不要代码块围栏、不要额外前言或结尾寒暄。"""

SYSTEM_PROMPT = (
    "你是企业会议纪要助手，根据会议转写生成中文会议纪要（Markdown）。"
    "只能依据转写内容，禁止编造。\n\n"
    "输出骨架（严格遵循）：\n\n"
    f"{MINUTES_SKELETON}\n\n"
    f"{MINUTES_RULES}"
)


def _llm_config() -> tuple[str, str, str]:
    base_url = os.getenv("OPENAI_BASE_URL", "https://api.deepseek.com/v1")
    api_key = os.getenv("OPENAI_API_KEY", "")
    model = os.getenv("OPENAI_MODEL", "deepseek-chat")
    return base_url, api_key, model


def summarize_meeting(*, title: str, attendees: list[str], transcript: str) -> dict[str, Any]:
    base_url, api_key, model = _llm_config()
    if not api_key or os.getenv("MOCK_LLM", "").lower() in ("1", "true", "yes"):
        log.warning("using mock LLM summary (no OPENAI_API_KEY or MOCK_LLM=1)")
        return {
            "summary": (
                f"## 会议概览\n\n- **会议主题**：《{title or '未命名会议'}》\n"
                f"- **会议摘要**：mock 纪要（未配置 OPENAI_API_KEY 时的测试占位）。\n\n"
                f"## 待办事项\n\n- [ ] 配置 OPENAI_API_KEY 后重新生成真实纪要\n\n"
                f"<!-- 转写预览：{transcript[:200]} -->"
            ),
            "todos": ["（mock）请配置 OPENAI_API_KEY 后重新生成真实纪要"],
            "decisions": [],
            "risks": [],
            "model": "mock",
        }

    # 用纯文本调用而非 with_structured_output：OpenAI 兼容端点对 response_format 支持不一
    # （DeepSeek 该模型会返回 400 This response_format type is unavailable now），
    # 而纪要正文本身就是一段 Markdown，骨架由提示词约束即可。
    llm = ChatOpenAI(base_url=base_url, api_key=api_key, model=model, temperature=0.2)
    messages = [
        SystemMessage(content=SYSTEM_PROMPT),
        HumanMessage(
            content=(
                f"会议标题：{title or '未命名会议'}\n"
                f"参会人：{'、'.join(attendees) if attendees else '未知'}\n\n"
                f"转写全文：\n{transcript}"
            )
        ),
    ]
    msg = llm.invoke(messages)
    minutes = _strip_code_fence((getattr(msg, "content", "") or "").strip())
    if not minutes:
        raise RuntimeError("模型返回了空的会议纪要")
    log.info("summarize done model=%s minutes_len=%d", model, len(minutes))
    return {
        "summary": minutes,
        # 纪要正文已包含决议/待办/风险章节，结构化字段留空以兼容旧前端与历史数据。
        "todos": [],
        "decisions": [],
        "risks": [],
        "model": model,
    }


def _strip_code_fence(text: str) -> str:
    """模型偶尔会把 Markdown 包在 ``` 围栏或 JSON 里，这里做一次容错。"""
    if text.startswith("{"):
        start, end = text.find("{"), text.rfind("}")
        if start >= 0 and end > start:
            try:
                data = json.loads(text[start : end + 1])
                if isinstance(data, dict) and isinstance(data.get("minutes"), str):
                    text = data["minutes"].strip()
            except json.JSONDecodeError:
                pass
    if text.startswith("```"):
        lines = text.splitlines()
        if lines and lines[0].startswith("```"):
            lines = lines[1:]
        if lines and lines[-1].strip() == "```":
            lines = lines[:-1]
        text = "\n".join(lines).strip()
    return text
