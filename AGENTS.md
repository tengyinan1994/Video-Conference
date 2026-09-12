# Video Conference — 项目协作指南（Agent 入口）

私有化视频会议 Demo：**HotGo 业务底座 + LiveKit 媒体 + 独立 Vue3 会议客户端**。

本文件只负责**跨工程**的信息：仓库拓扑、启动方式、改动落点、易踩坑。
各子工程细节**不在此重复**，按下表跳转。

---

## ⚠️ 生产环境状态（先读这一节）

> **`deploy/prod/` 是生产环境，已上线，有真实用户在用。**
> 目录名已从 `test/` 更名为 `prod/`，不存在"可以随便部署"的环境。

- `deploy/prod/` = **生产（PROD）**，远端 `dept`，`PUBLIC_HOST=125.211.217.19`。
- `deploy/dev/` = 唯一可以自由折腾的环境（本机）。
- **Agent 不得自主部署**：不能以"验证一下""顺手部署"为由执行 `deploy.sh` 的任何子命令。
  只有用户**在当轮对话中明确要求**部署时才可以执行；执行前先复述影响范围并等确认。
- **一次只动一个服务**：优先 `update <service>`，**绝不用 `full`**（会重建并重启全部服务）。
- **禁止**：`down`（停掉全部生产服务）、改动远端 `/data/video-conference/data/`、删除或重命名它。
- 影响生产的改动（库结构、配置、容器编排）必须先讲清风险，再等用户确认。

### 为什么：一条真实的数据丢失路径

`deploy/prod/docker-compose.yml:24-31` 把 `init/*.sql` 挂到 MySQL 的
`/docker-entrypoint-initdb.d/`。该目录**只在数据目录为空时执行**：

- 正常情况：`${DATA_DIR}/mysql` 已有数据 → 初始化脚本不执行，安全。
- **危险情况**：若 `DATA_DIR` 配错、指向空目录或被清空，MySQL 会判定为"首次初始化"，
  重新执行建表与初始化 SQL —— 生产数据直接受损。

所以：**不要改 `deploy/prod/.env` 里的 `DATA_DIR` / `REMOTE_DIR`，不要动远端 `data/` 目录。**

（附带发现：`09-conference-minutes.sql`、`10-...permission.sql` **没有**挂进 compose，
即从零初始化得到的并不是完整库，增量 SQL 需人为执行 —— 这更说明不要指望"重建数据目录"。）

### 改生产功能的正确顺序

1. 在本机 `deploy/dev/` 验证通过
2. 库变更：先写增量 SQL，**备份后**再在生产手工执行
   （备份：`rsync -a dept:/data/video-conference/data/ ./backup/`）
3. 代码变更：只 `update` 受影响的那一个服务
4. 立刻确认：`./deploy/prod/deploy.sh status` + `logs <service>`，并给出可复现的验收路径

---

## 指令优先级

| 作用范围 | 文件 |
|---|---|
| 仓库总体 | `AGENTS.md`（本文件） |
| `server/` HotGo 后端 + 管理后台 | [`server/AGENTS.md`](server/AGENTS.md)（分层 / CRUD / 字典 / 插件 / 日志等强规范，**改动前必读**） |
| `client/` 会议客户端 | [`client/AGENTS.md`](client/AGENTS.md) |
| 子工程内部文档 | `server/README.md`、`client/README.md`、`deploy/dev/README.md`、`deploy/prod/README.md`、`services/minutes-worker/README.md` |

更具体的文件优先。`server/AGENTS.md` 源自 HotGo 上游，**不适用于 `client/`**。

## 仓库拓扑

| 路径 | 内容 | 技术栈 |
|---|---|---|
| `client/` | 会议客户端（网页 + Tauri 2 桌面，同一套 `src/`） | Vue 3.5 + Ant Design Vue 4 + livekit-client + Vite 8 + TS |
| `server/backend/` | HotGo 后端 | Go 1.26 + GoFrame |
| `server/backend/addons/conference/` | **会议业务全部逻辑**（Token / 会议 / 录制 / 纪要 / 认证） | HotGo Addon 模式 |
| `server/web/` | 管理后台 | Vue 3 + Naive UI + Pinia + axios（HotGo 上游） |
| `services/minutes-worker/` | 会后 AI 纪要 Worker | Python + FastAPI + FunASR/mock ASR + LangChain |
| `deploy/dev/` | 本机开发中间件（LiveKit / RustFS / Egress） | docker compose |
| `deploy/prod/` | **生产环境**（dept，已上线）全量部署（含 `init/*.sql` 增量脚本）⚠️ 见文首 | docker compose + 一键脚本 |
| `deploy/images/` | 镜像打包 | — |
| `livekit-master/` | LiveKit 上游源码，**只读参考，已被 .gitignore 忽略** | Go |

**改动落点速查**

- 会议功能（前后端）→ `client/src/` + `server/backend/addons/conference/`
- 管理后台的会议管理页 → `server/web/src/` + 上述 addon 的 `api/admin`、`controller/admin`
- 不要动 `server/backend/internal/`（HotGo 框架层），除非确实要改框架行为
- 不要把会议业务写进 `server/web/` 或 HotGo 主模块

## 本机开发（macOS）

**本节全部操作只作用于本机 `deploy/dev/`，不涉及生产。** 生产见文首警告与文末「部署」。

依赖：Go + air、pnpm、Docker、Python 3（仅纪要 Worker 需要）。
MySQL / Redis 复用本机 Middleware（如 `/Users/chaoming/Middleware/docker-compose.yml`），独立库 `video_conference`。

```bash
# 1. LiveKit（7880 信令 / 7881 / 7882）；换网后先 ./deploy/dev/ip.sh -w 再重跑
./deploy/dev/dev-livekit.sh

# 2. HotGo 后端（:8000）
cd server/backend && air

# 3. 会议客户端（https://127.0.0.1:5173）
cd client && pnpm dev

# 4. 管理后台
cd server/web && pnpm dev

# 5. 录制（可选）：先起 RustFS，再起 Egress
docker compose -f deploy/dev/docker-compose.yml --env-file deploy/dev/.env up -d rustfs
./deploy/dev/start-egress-when-ready.sh

# 6. 会后 AI 纪要（可选）：见 services/minutes-worker/README.md
```

VS Code：`.vscode/tasks.json` 提供 `启动全部` / `仅开会（不含管理后台）` 一键任务。

**本机端口**

| 端口 | 用途 |
|---|---|
| 5173 | 会议客户端（HTTPS 自签） |
| 8000 | HotGo 后端 API（`/api.json`、`/swagger`） |
| 8001 | 管理后台（`server/web/.env` 的 `VITE_PORT`，代理 `/api` → 8000） |
| 7880 / 7881 / 7882 | LiveKit 信令 / RTC TCP / RTC UDP |
| 17886 / 17887 | RustFS S3 / 控制台（开发） |
| 8090 | minutes-worker（`PORT` 可覆盖） |

## 跨工程约定

- **代理**：客户端开发态走 Vite 代理 —— `/api` → `127.0.0.1:8000`，`/rtc` → `127.0.0.1:7880`（同源 WS，规避 Cursor 内置预览拦截直连）。换网只改 `deploy/dev/.env`，通常不用改后端 `livekit.url`。
- **HTTPS 强制**：纯网页开发必须 `https`（`@vitejs/plugin-basic-ssl`），否则摄像头 / 麦克风 / 屏幕共享不可用；Tauri 壳走 `http://localhost`（安全上下文），**不要**给 Tauri 开自签证书。
- **两套前端互不复用**：`client/` 用 Ant Design Vue + 自封装 `fetch`（`src/utils/request.ts`）；`server/web/` 用 Naive UI + axios。**不要**把 `server/web` 的请求层或组件搬进 `client/`。
- **响应格式**：HotGo 统一 `{code, message, data}`，`code === 0` 为成功；客户端 `401/61` 会清登录态并跳登录页。
- **数据库改动**：增量脚本放 `deploy/prod/init/NN-xxx.sql`（已有 01~10），**只加新文件、不改历史脚本**；表名 `hg_addon_conference_*`。⚠️ 这些脚本只对**空数据目录**自动执行，生产上必须手工在备份后执行（见文首）。
- **配置分层**：`manifest/config/config.example.yaml` 可提交，`config.yaml` 已 gitignore（`server/backend/.gitignore:17`），本地密钥只写后者。
- **密钥与 DSN 不入库**：`.cursor/dbhub.env`、`.env`、`deploy/*/.env`、`deploy/prod/config/config.yaml` 均被忽略，勿提交、勿写入文档。

## 会议业务地图

- 入口 API：`POST /api/conference/token/create`（body `{"room","nickname"}`）；代码在 `addons/conference/api/api/token/`。
- 路由：`router/api.go`（会议端）+ `router/admin.go`（管理端）+ `router/genrouter/`。
- 客户端路由：`/login`、`/` 大厅、`/join/:shareCode` 游客加入、`/room/:room` 会议室、`/egress/:room?` 录制模板页（无需登录）。
- 角色：参与者 metadata 里 `host` / `member`（`consts/conference.go`）；主持人标记有 2h 缓存。
- 录制：`recording_purpose` 区分 `playback`（进回放列表）与 `ai`（仅喂纪要、不进列表）；合成默认 2K@60。
- 会后纪要：状态机 `pending → transcribing → summarizing → ready|failed`；全程静音落 `skipped_empty`。Worker 不可用或 `minutes.enabled=false` 时**不得影响会议本身**。
- 管理后台回放/下载都经 HotGo HTTPS 代理（`/admin/conference/recording/play|download`），浏览器**不直连** RustFS；`recording.publicEndpoint` 保持为空。

## 部署（= 生产操作，需用户明确授权）

**`deploy/prod/deploy.sh` 的每一条子命令都在动生产。** 首次配置 (`cp ... .env`) 已完成，勿重复覆盖。

| 命令 | 影响 | 评价 |
|---|---|---|
| `update <service>` | 重建并重启**单个**服务，其余不动 | ✅ 唯一的常规发布方式 |
| `status` / `logs` | 只读 | ✅ 随时可查 |
| `sync` | 同步 compose/init/config/`.env` 到生产 | ⚠️ 会覆盖远端配置，改动前先 diff |
| `pull` | 拉公共镜像 | ⚠️ 会拉新版本，可能改变运行行为 |
| `full` | 重建**全部**镜像并重启全部服务 | ❌ 生产禁用 |
| `down` | 停掉**全部**服务 = 用户断会 | ❌ 生产禁用 |

```bash
# 常规发布（只动一个服务）
./deploy/prod/deploy.sh update hotgo|client|admin|livekit|minutes-worker
./deploy/prod/deploy.sh status
./deploy/prod/deploy.sh logs hotgo
```

会议页与管理后台**分端口 HTTPS**（默认 17885 / 17883），自签证书需手动信任。dept 上 MySQL / Redis / HotGo / minutes-worker / Egress / RustFS 均**不映射宿主机**，只走 compose 内网 `vc_net`。

**注意 `update` 仍会造成该服务的短暂中断**（`--force-recreate`）。发布前先确认当前没有进行中的会议。

## 工程习惯

- **提交信息**：`<gitmoji>? <type>(<scope>): <中文描述>`，如 `✨ feat(minutes): 添加会后 AI 纪要功能与相关接口`；scope 用 `meeting` / `recording` / `deploy` / `conference` 等模块名。保持中文描述。
- **验收习惯**：每个功能给出可复现的验证路径（起哪些服务、点什么、看什么结果），而不只是"改完了"。
- **文档同步**：改启动方式、端口、部署步骤时，同步更新 `README.md` 与对应子工程文档。
- **测试**：`addons/conference` 下已有 `*_test.go`（token/identity/meeting/recording）；改这些逻辑时跑 `cd server/backend && go test ./addons/conference/...`。

## 记忆与上下文

- 本项目已接入 **OpenViking 长期记忆**。开工前若涉及历史决策（"上次怎么改的""为什么这么做"），先查记忆再动手。
- 值得长期保留的结论（架构决策、踩坑原因、环境约定）写入记忆，而不是只留在对话里；敏感信息（密钥、密码、DSN）**不要**写入记忆。
