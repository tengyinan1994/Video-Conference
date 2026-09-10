FROM python:3.11-slim-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements-runtime.txt requirements-funasr.txt ./
RUN pip install --no-cache-dir -r requirements-runtime.txt \
    && pip install --no-cache-dir -r requirements-funasr.txt

COPY main.py asr.py summarize.py ./

ENV ASR_BACKEND=funasr
ENV MOCK_LLM=0
ENV PORT=8090
ENV FUNASR_MODEL=paraformer-zh
# ModelScope / FunASR 缓存
ENV MODELSCOPE_CACHE=/data/models
ENV HOME=/data

EXPOSE 8090
CMD ["python", "-m", "uvicorn", "main:APP", "--host", "0.0.0.0", "--port", "8090"]
