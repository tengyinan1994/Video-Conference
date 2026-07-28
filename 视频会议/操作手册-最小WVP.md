# 操作手册：最小 WVP（阶段 1～4）

> 对应 [[学习和实现路线]] 已完成的阶段 1～4。  
> 本文只讲**怎么起、怎么改地址、怎么打安装包**，不讲原理（原理见 [[阶段1-2亲自懂]]）。

当前形态：**HotGo 发 Token + LiveKit 传媒体 + Vue3 会议页**，同一套前端可网页打开，也可打成 Win/Mac 桌面壳（Tauri 2）。没有登录、预约、录制、一键部署包——那些是阶段 5。

---

## 0. 三块服务、两套地址

```
客户端（浏览器 / 安装包）
    │
    │  ① HTTP：拿 Token、踢人、全员静音
    ▼
HotGo :8000  ─── livekit.url 写进 Token 响应里的 serverUrl
    │
    │  ② WebSocket + WebRTC：进房、音视频
    ▼
LiveKit :7880（信令）/ :7881 TCP / :7882 UDP（媒体）
```

| 角色 | 默认端口 | 谁连它 | 配置写在哪 |
|------|----------|--------|------------|
| HotGo API | `8000` | 客户端发 HTTP | `client/.env.production` 的 `VITE_API_BASE_URL`（安装包写死） |
| LiveKit | `7880` 等 | 客户端进房 | HotGo `config.yaml` 的 `livekit.url`（接口返回给客户端） |
| 会议页开发服 | `5173` | 本机浏览器 / Tauri dev | 开发期 API 走 Vite 代理，一般不用写死 |

记住：**业务地址**和 **媒体地址**是两套，改完一边不等于另一边自动对。

---

## 1. 本机开发怎么起

### 1.1 依赖

- MySQL + Redis（本机中间件即可；库名 `video_conference`）
- LiveKit（`brew install livekit` 或 Docker，见下）
- Go + Air（HotGo）
- Node + pnpm（客户端）
- 桌面壳额外：Rust（`rustup`）、macOS 还要 Xcode Command Line Tools

### 1.2 三个进程

```bash
# 终端 1：LiveKit
# 本机双开验证可用：
livekit-server --dev
# 或局域网第二台设备：用仓库脚本（会自动填本机局域网 IP 作 ICE）
zsh scripts/start-livekit.sh

# 终端 2：HotGo
cd server/backend && air

# 终端 3：会议客户端（二选一）
cd client && pnpm install
pnpm dev          # 网页 → https://127.0.0.1:5173（自签证书）
pnpm tauri:dev    # 桌面壳（会自己拉 Vite，勿再开一份占 5173）
```

Cursor / VS Code：复合启动「仅开会（不含管理后台）」或「仅开会（Tauri 壳）」。

开发期约定：

- `client/.env.development` 里 `VITE_API_BASE_URL=` **空** → 请求走相对路径 `/api/...`
- Vite 把 `/api` 代理到 `http://127.0.0.1:8000`，把 `/rtc` 代理到 LiveKit（方便内置预览）
- 网页模式开了自签 https（摄像头/麦需要安全上下文）；Tauri 壳连 `http://localhost:5173`，不开 https

---

## 2. 地址怎么配（最容易踩坑）

换机器、换网段、打安装包给别人用之前，**只认下面三处**。

### 2.1 HotGo → LiveKit（媒体地址）

文件：`server/backend/manifest/config/config.yaml`

```yaml
livekit:
  url: "ws://10.64.3.83:7880"   # ← 客户端拿到的 serverUrl，必须是「对方机器能连上的地址」
  apiKey: "devkey"
  apiSecret: "secret"
  tokenTTL: 900
  allowAnonymousToken: true
  rateLimitPerMinute: 30
```

| 场景 | `livekit.url` 填什么 |
|------|----------------------|
| 只本机双开 | `ws://127.0.0.1:7880` |
| 局域网多人 / 安装包 | `ws://<HotGo那台机器的局域网IP>:7880`，例如 `ws://10.64.3.83:7880` |
| 以后有 TLS | `wss://会议域名:443`（需证书，阶段 5 再正规化） |

改完 **重启 HotGo**（Air 一般会热更配置；不确定就重启一次）。

客户端进房时用的是接口返回的 `serverUrl`，**不是**前端再写一份 LiveKit 地址。  
例外：纯 `pnpm dev` 网页开发时，前端会把信令改走同源 `/rtc` 代理（见 `resolveLiveKitUrl`）；**安装包 / `pnpm build` 产物不走这条捷径**，完全听 HotGo 返回的 URL。

### 2.2 客户端 → HotGo（业务地址）

文件：`client/.env.production`

```bash
# 安装包 / 生产构建写死；开发 pnpm dev 不受此项影响
VITE_API_BASE_URL=http://10.64.3.83:8000
```

| 场景 | 怎么填 |
|------|--------|
| 本机 `pnpm dev` / `tauri:dev` | 不用管；用 `.env.development` 空值 + 代理 |
| 打桌面安装包、或 `pnpm build` 静态页 | 填局域网可达的 HotGo，如 `http://10.64.3.83:8000`（不要末尾 `/`） |
| 换 IP | **改这个文件后重新打包**；已发出去的安装包不会跟着变 |

Vite 在 build 时把 `VITE_*` 打进 JS。装到别人电脑上的客户端，不会读你本机的 `.env`。

### 2.3 LiveKit 的 ICE 宣告 IP（局域网第二台设备）

信令连上了、几秒后断、或只有本机有画面：多半是 LiveKit 把媒体地址宣告成了 `127.0.0.1` / Docker 内网 IP。

用仓库脚本时会自动探测本机局域网 IP 并写入 `node_ip`：

```bash
zsh scripts/start-livekit.sh
# 日志里会看到：LiveKit ICE node_ip -> 10.x.x.x
```

自己跑 Docker 时至少保证：

```yaml
rtc:
  use_external_ip: false
  node_ip: <本机局域网IP>   # 别的设备能 ping 通的那个
```

防火墙放行：`7880`、`7881/tcp`、`7882/udp`。

本机 IP 变了（换 Wi‑Fi）：脚本会发现不一致并重建容器；HotGo 的 `livekit.url` 也要改成新 IP。

### 2.4 一页对照（交付前自检）

假设服务器局域网 IP 是 `10.64.3.83`：

| 配置项 | 期望值 |
|--------|--------|
| HotGo `server.address` | `:8000`（监听所有网卡） |
| HotGo `livekit.url` | `ws://10.64.3.83:7880` |
| LiveKit `node_ip` | `10.64.3.83` |
| `client/.env.production` | `VITE_API_BASE_URL=http://10.64.3.83:8000` |
| 客户端连业务 | `http://10.64.3.83:8000/api/conference/...` |
| 客户端连媒体 | `ws://10.64.3.83:7880` |

本机验证：浏览器打开 `http://10.64.3.83:8000/api.json`（或任意健康检查）能通；另一台设备 `ping 10.64.3.83` 通，再开客户端进同一房间。

---

## 3. 打包

### 3.1 网页静态资源（阶段 5 交付包仍会用）

```bash
cd client
# 先确认 .env.production 里的 VITE_API_BASE_URL
pnpm build
```

产物：`client/dist/`。可用任意静态服务器托管；**必须能访问到** `.env.production` 里写的 HotGo。  
局域网用 `http://内网IP` 打开静态页时，浏览器可能因非安全上下文拦摄像头——真机开会优先用 **桌面安装包**，或后面上 https。

### 3.2 桌面安装包（Tauri 2）

```bash
cd client
# 1) 改好 .env.production（HotGo 地址）
# 2) 确认 HotGo config.yaml 里 livekit.url 是局域网地址
pnpm tauri:build
```

产物目录：

```
client/src-tauri/target/release/bundle/
├── dmg/          # macOS 安装盘
├── macos/        # .app
├── msi/ / nsis/  # Windows（在 Windows 上编才会出）
└── ...
```

说明：

- **在哪台机器编，就出哪台的包**：Mac 上编出 `.dmg` / `.app`；Windows 安装包要在 Windows 上编（或以后配 CI）。
- 首次编译会拉 Rust 依赖，较慢；之后增量快很多。
- 当前未做代码签名：macOS 首次打开可能要「系统设置 → 隐私与安全性」里允许；屏幕录制/摄像头/麦克风按系统弹窗授权（`Info.plist` 已写用途说明）。
- Windows：需 WebView2 Runtime（Win10/11 一般自带）。

改业务地址或产品名后必须重新 `pnpm tauri:build`。产品名 / 包标识在 `client/src-tauri/tauri.conf.json`（`productName`、`identifier`、`version`）。

### 3.3 安装包验收清单

- [ ] 安装后能打开，输入房间名 + 昵称可进会
- [ ] 与网页端或另一台桌面端进同一房间，能看能听
- [ ] 开关麦 / 摄像头 / 屏幕共享正常（macOS 投屏要在「屏幕录制」权限里勾选本应用）
- [ ] 断网提示可读；HotGo 关掉时进房失败提示指向业务服务

---

## 4. 常见问题（按层排查）

| 现象 | 先查哪一层 |
|------|------------|
| 进房页提示连不上业务服务 / HTTP 失败 | `VITE_API_BASE_URL` 或 HotGo 没起；安装包是否用了旧 IP |
| 能拿 Token，但进房失败 / 马上断开 | `livekit.url`、LiveKit 进程、防火墙、ICE `node_ip` |
| 本机双开有画面，第二台设备没有 | LiveKit `node_ip` 仍是 `127.0.0.1`；或第二台连的 `serverUrl` 不可达 |
| 桌面端无摄像头/麦 | 系统隐私权限；杀毒软件拦截 |
| `tauri:dev` 起不来 / 5173 占用 | 先关掉单独的 `pnpm dev` |
| 网页 `http://内网IP:5173` 开不了摄像头 | 非安全上下文；用 https 开发服或改用安装包 |

分层口诀（和路线图一致）：

1. Vue / LiveKit 客户端（`client/src`）  
2. 壳 / 权限（`src-tauri`，仅桌面）  
3. Token / 会控（HotGo `conference` addon）  
4. LiveKit 服务端 / 网络

---

## 5. 和阶段 5 的边界（别期待现在就有）

本手册覆盖的「最小 WVP」**没有**：

- 登录、会议室预约、管理后台看会  
- Egress 录制、MinIO  
- docker compose 一键部署全家桶  
- 安装包内可改服务器地址的设置页 / 自动更新  

换环境现在的做法就是：**改配置 → 重启 HotGo / LiveKit → 必要时重打安装包**。阶段 5 再做成可交付运维形态。

---

## 6. 相关文件速查

| 文件 | 用途 |
|------|------|
| `client/.env.production` | 安装包 / 生产构建的 HotGo 根地址 |
| `client/.env.development` | 开发用；保持空 base 走代理 |
| `client/vite.config.ts` | 开发代理 `/api`、`/rtc`；Tauri 时关自签 https |
| `client/src/utils/apiBase.ts` | 读 `VITE_API_BASE_URL` |
| `server/backend/manifest/config/config.yaml` | HotGo 端口 + `livekit.*` |
| `scripts/start-livekit.sh` | Docker LiveKit + 自动局域网 `node_ip` |
| `client/src-tauri/tauri.conf.json` | 窗口、图标、打包目标 |
| `client/src-tauri/Info.plist` | macOS 摄像头/麦/录屏说明 |
| `.vscode/launch.json` | 一键起 LiveKit + HotGo + Client |
