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

**安装包**：生产 API 写在 `client/.env.production`（`VITE_API_BASE_URL=https://125.211.217.19:17885`）。换地址时改该文件后重新 `pnpm tauri:build`。局域网开会时 HotGo 的 `livekit.url` 和 LiveKit `node_ip` 要用对端能访问的地址，不要用 `127.0.0.1`。

`:8000` = 本机开发 HotGo（发 Token / 踢人 / 全员静音）；`:7880` = 本机 LiveKit 媒体。

## 目录

- `src/`：会议 UI 与 LiveKit（Web / 桌面共用）
- `src/utils/platform.ts`：`isTauri()` 环境判断
- `src-tauri/`：**Rust 壳**（窗口、系统权限、打包）；读懂级即可
- `src/views/JoinView.vue`：入会
- `src/views/RoomView.vue`：会议页
- `src/composables/useLiveKitRoom.ts`：唯一持有 LiveKit `Room`
- `src/api/conference.ts`：调用 `POST /api/conference/token/create`

## 打包桌面端（Windows NSIS）

当前只打 **Windows NSIS `setup.exe`**（`src-tauri/tauri.conf.json` 的 `bundle.targets` = `nsis`）。**真正产出安装包需要一台 Windows 机器**；在 macOS / Linux 上跑 `pnpm tauri:build` 不会生成 NSIS。

生产 API 由 Vite 在 `pnpm build` 时读入 `client/.env.production`（`VITE_API_BASE_URL=https://125.211.217.19:17885`）。仓库里同时有同内容的 `.env.production.example`。换地址后重新打包即可。

```bash
# 在 Windows 上
cd client
pnpm install
pnpm tauri:build
```

产物在 `src-tauri/target/release/bundle/nsis/`（`*setup.exe`）。安装程序会把生产自签证书写入当前用户的受信任根存储；轮换证书时需替换 `src-tauri/windows/prod-17885.cer` 后重新打包。安装后冒烟：登录 → 大厅 → 加入会议 → 音视频（打到生产 HotGo）。

LiveKit 信令解析单测（不依赖 Windows）：

```bash
cd client
pnpm test
```
