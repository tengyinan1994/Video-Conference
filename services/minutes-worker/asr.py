"""ASR：FunASR（默认）或 mock。"""

from __future__ import annotations

import logging
import os
import subprocess
import threading
from pathlib import Path

log = logging.getLogger("minutes-worker.asr")

ASR_BACKEND = os.getenv("ASR_BACKEND", "funasr").lower()  # funasr | mock

_model = None
_model_lock = threading.Lock()


def _to_wav(src: Path, dest: Path) -> Path:
    dest.parent.mkdir(parents=True, exist_ok=True)
    cmd = [
        "ffmpeg",
        "-y",
        "-i",
        str(src),
        "-ac",
        "1",
        "-ar",
        "16000",
        str(dest),
    ]
    proc = subprocess.run(cmd, capture_output=True)
    if proc.returncode != 0:
        err = (proc.stderr or b"").decode("utf-8", errors="replace")[:500]
        raise RuntimeError(f"ffmpeg 转码失败: {err}")
    return dest


def _get_funasr_model():
    global _model
    if _model is not None:
        return _model
    with _model_lock:
        if _model is not None:
            return _model
        from funasr import AutoModel

        model_id = os.getenv("FUNASR_MODEL", "paraformer-zh")
        vad = os.getenv("FUNASR_VAD_MODEL", "fsmn-vad")
        punc = os.getenv("FUNASR_PUNC_MODEL", "ct-punc")
        log.info("load FunASR model=%s vad=%s punc=%s", model_id, vad, punc)
        kwargs = {"model": model_id, "disable_update": True}
        if vad:
            kwargs["vad_model"] = vad
        if punc:
            kwargs["punc_model"] = punc
        _model = AutoModel(**kwargs)
        return _model


def transcribe_files(paths: list[Path]) -> str:
    if ASR_BACKEND == "mock":
        if os.getenv("MOCK_EMPTY", "").lower() in ("1", "true", "yes"):
            return ""
        names = ", ".join(p.name for p in paths)
        return (
            f"【mock 转写】共 {len(paths)} 段音源（{names}）。"
            "主持人确认下周交付纪要功能，待办：完成 FunASR 接入、大厅展示纪要。"
        )

    if not paths:
        return ""

    wavs: list[Path] = []
    work = paths[0].parent / "wav"
    for i, p in enumerate(paths):
        wav = work / f"{i}.wav"
        _to_wav(p, wav)
        wavs.append(wav)

    model = _get_funasr_model()
    parts: list[str] = []
    for wav in wavs:
        log.info("funasr generate input=%s", wav)
        res = model.generate(input=str(wav))
        text = _extract_text(res)
        log.info("funasr text_len=%d preview=%r", len(text), text[:80])
        if text:
            parts.append(text)
    return "\n".join(parts).strip()


def _extract_text(res) -> str:
    if res is None:
        return ""
    if isinstance(res, str):
        return res.strip()
    if isinstance(res, dict):
        return str(res.get("text") or res.get("preds") or "").strip()
    if isinstance(res, list) and res:
        first = res[0]
        if isinstance(first, dict):
            return str(first.get("text") or first.get("preds") or "").strip()
        return str(first).strip()
    return str(res).strip()
