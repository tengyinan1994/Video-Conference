"""会后 AI 纪要 Worker：拉 AI 音源 → ASR → LangChain 总结 → 回调 HotGo。"""

from __future__ import annotations

import hashlib
import hmac
import logging
import os
import tempfile
import threading
from pathlib import Path
from typing import Any

import boto3
import httpx
from botocore.client import Config
from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel, Field

from asr import transcribe_files
from summarize import summarize_meeting

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("minutes-worker")

APP = FastAPI(title="conference-minutes-worker", version="0.1.0")

CALLBACK_URL = os.getenv("HOTGO_CALLBACK_URL", "http://127.0.0.1:8000/api/conference/minutes/callback")
SECRET = os.getenv("MINUTES_WORKER_SECRET", "")
S3_ENDPOINT = os.getenv("S3_ENDPOINT", "http://127.0.0.1:17886")
S3_ACCESS_KEY = os.getenv("S3_ACCESS_KEY", "rustfsadmin")
S3_SECRET_KEY = os.getenv("S3_SECRET_KEY", "rustfsadmin")
S3_BUCKET = os.getenv("S3_BUCKET", "recordings")
S3_REGION = os.getenv("S3_REGION", "us-east-1")
S3_FORCE_PATH = os.getenv("S3_FORCE_PATH_STYLE", "true").lower() in ("1", "true", "yes")


class Segment(BaseModel):
    id: int
    seq: int = 0
    objectKey: str
    fileSize: int = 0


class JobRequest(BaseModel):
    meetingId: int
    title: str = ""
    attendees: list[str] = Field(default_factory=list)
    segments: list[Segment] = Field(default_factory=list)
    sourceRecordingIds: list[int] = Field(default_factory=list)
    minTranscriptChars: int = 8


def sign_body(body: bytes) -> str:
    if not SECRET:
        return ""
    mac = hmac.new(SECRET.encode("utf-8"), body, hashlib.sha256)
    return mac.hexdigest()


def verify_incoming(sig: str | None, body: bytes) -> bool:
    if not SECRET:
        return True
    if not sig:
        return False
    expect = sign_body(body)
    return hmac.compare_digest(expect, sig.strip())


def s3_client():
    return boto3.client(
        "s3",
        endpoint_url=S3_ENDPOINT,
        aws_access_key_id=S3_ACCESS_KEY,
        aws_secret_access_key=S3_SECRET_KEY,
        region_name=S3_REGION,
        config=Config(signature_version="s3v4", s3={"addressing_style": "path" if S3_FORCE_PATH else "auto"}),
    )


def callback(payload: dict[str, Any]) -> None:
    import json

    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    sig = sign_body(body)
    if sig:
        headers["X-Minutes-Signature"] = sig
    try:
        with httpx.Client(timeout=60.0) as client:
            r = client.post(CALLBACK_URL, content=body, headers=headers)
            if r.status_code >= 300:
                log.warning("callback failed status=%s body=%s", r.status_code, r.text[:300])
            else:
                log.info("callback ok meeting=%s status=%s", payload.get("meetingId"), payload.get("status"))
    except Exception as e:
        log.exception("callback error: %s", e)


def download_segments(segments: list[Segment], work: Path) -> list[Path]:
    client = s3_client()
    paths: list[Path] = []
    for seg in sorted(segments, key=lambda s: s.seq):
        key = seg.objectKey.strip()
        if not key or "{" in key:
            continue
        suffix = Path(key).suffix or ".ogg"
        dest = work / f"{seg.seq}_{seg.id}{suffix}"
        log.info("download s3://%s/%s -> %s", S3_BUCKET, key, dest)
        client.download_file(S3_BUCKET, key, str(dest))
        paths.append(dest)
    return paths


def run_job(job: JobRequest) -> None:
    meeting_id = job.meetingId
    source_ids = job.sourceRecordingIds or [s.id for s in job.segments]
    try:
        callback(
            {
                "meetingId": meeting_id,
                "status": "transcribing",
                "sourceRecordingIds": source_ids,
            }
        )
        with tempfile.TemporaryDirectory(prefix="minutes-") as tmp:
            work = Path(tmp)
            paths = download_segments(job.segments, work)
            if not paths:
                callback(
                    {
                        "meetingId": meeting_id,
                        "status": "unavailable",
                        "errorMsg": "无可用的 AI 音源文件",
                        "sourceRecordingIds": source_ids,
                    }
                )
                return

            transcript = transcribe_files(paths)
            transcript = (transcript or "").strip()
            min_chars = max(1, job.minTranscriptChars)
            if len(transcript) < min_chars:
                callback(
                    {
                        "meetingId": meeting_id,
                        "status": "skipped_empty",
                        "transcript": transcript,
                        "errorMsg": "未检测到有效发言",
                        "sourceRecordingIds": source_ids,
                    }
                )
                return

            callback(
                {
                    "meetingId": meeting_id,
                    "status": "summarizing",
                    "transcript": transcript,
                    "sourceRecordingIds": source_ids,
                }
            )
            result = summarize_meeting(
                title=job.title,
                attendees=job.attendees,
                transcript=transcript,
            )
            callback(
                {
                    "meetingId": meeting_id,
                    "status": "ready",
                    "transcript": transcript,
                    "summary": result.get("summary", ""),
                    "structured": {
                        "todos": result.get("todos", []),
                        "decisions": result.get("decisions", []),
                        "risks": result.get("risks", []),
                    },
                    "sourceRecordingIds": source_ids,
                    "model": result.get("model", ""),
                    "errorMsg": "",
                }
            )
    except Exception as e:
        log.exception("job failed meeting=%s", meeting_id)
        callback(
            {
                "meetingId": meeting_id,
                "status": "failed",
                "errorMsg": str(e)[:500],
                "sourceRecordingIds": source_ids,
            }
        )


@APP.get("/healthz")
def healthz():
    return {"ok": True}


@APP.post("/v1/jobs")
async def create_job(request: Request):
    body = await request.body()
    sig = request.headers.get("X-Minutes-Signature")
    if not verify_incoming(sig, body):
        raise HTTPException(status_code=401, detail="bad signature")
    try:
        job = JobRequest.model_validate_json(body)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"bad json: {e}") from e
    if not job.segments:
        raise HTTPException(status_code=400, detail="segments required")
    threading.Thread(target=run_job, args=(job,), daemon=True).start()
    return {"ok": True, "meetingId": job.meetingId}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(APP, host="0.0.0.0", port=int(os.getenv("PORT", "8090")))
