# 会后 AI 纪要 Worker
#
# 本机开发（mock ASR，不装 FunASR）：
#   cd services/minutes-worker
#   python -m venv .venv && source .venv/bin/activate
#   pip install fastapi uvicorn httpx boto3 pydantic langchain-openai langchain-core
#   export ASR_BACKEND=mock
#   export OPENAI_API_KEY=sk-...
#   export OPENAI_BASE_URL=https://api.deepseek.com/v1
#   export OPENAI_MODEL=deepseek-flash
#   export HOTGO_CALLBACK_URL=http://127.0.0.1:8000/api/conference/minutes/callback
#   export MINUTES_WORKER_SECRET=dev-minutes-secret
#   export S3_ENDPOINT=http://127.0.0.1:17886
#   python main.py
#
# HotGo config.yaml：
#   minutes:
#     enabled: true
#     workerUrl: "http://127.0.0.1:8090"
#     callbackSecret: "dev-minutes-secret"
#     minTranscriptChars: 20
#
# 验收：
# - 不开回放录制，开一场会说话后结束 → 大厅详情应有纪要（依赖 AI audio egress + Worker）
# - 全程静音 → skipped_empty
# - minutes.enabled=false 或 Worker 未起 → 状态 failed/unavailable，会议本身不受影响
