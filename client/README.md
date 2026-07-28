# 会议客户端（client）

独立 **Vue 3** + TypeScript + Vite + Ant Design Vue + `livekit-client`。

阶段 4 已在本目录加入 **`src-tauri/`（Tauri 2）**：同一套前端，网页与桌面两种分发。**不要**把会议页做进 `server/web`。

## 启动

先确保：

1. LiveKit：`ws://localhost:7880`（`devkey` / `secret`）
2. HotGo：`cd ../server/backend && air`（`:8000`）
3. 本客户端二选一：

```bash
pnpm install

# 网页
pnpm dev
# 浏览器打开 https://127.0.0.1:5173（开发期自签证书）

# 桌面壳（会自动拉起 Vite，勿再单独开一份 pnpm dev 占 5173）
pnpm tauri:dev
```

开发期 `/api` 由 Vite 代理到 HotGo，无需手配 CORS。

VS Code / Cursor：`Client (Vite)` 或 `Client (Tauri)`；复合启动可用「仅开会（Tauri 壳）」。

**安装包**：业务地址写死在 `client/.env.production`（当前 `http://10.64.3.83:8000`，即 HotGo）。换内网 IP 时改该文件后重新 `pnpm tauri:build`。HotGo 的 `livekit.url` 也要用局域网可达地址。

`:8000` = HotGo（发 Token / 踢人 / 全员静音）；`:7880` = LiveKit 媒体。

## 目录

- `src/`：会议 UI 与 LiveKit（Web / 桌面共用）
- `src/utils/platform.ts`：`isTauri()` 环境判断
- `src-tauri/`：**Rust 壳**（窗口、系统权限、打包）；读懂级即可
- `src/views/JoinView.vue`：入会
- `src/views/RoomView.vue`：会议页
- `src/composables/useLiveKitRoom.ts`：唯一持有 LiveKit `Room`
- `src/api/conference.ts`：调用 `POST /api/conference/token/create`

## 打包桌面端

```bash
pnpm tauri:build
```

产物在 `src-tauri/target/release/bundle/`（macOS `.dmg` / `.app` 等）。
