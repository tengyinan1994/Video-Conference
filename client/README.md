# 会议客户端（client）

独立 Vue3 + TypeScript + Vite + Ant Design Vue + `livekit-client`。

后续阶段 4 在本目录加入 `src-tauri/`，套成 Tauri 桌面客户端；**不要**把会议页做进 `server/web`。

## 启动

先确保：

1. LiveKit：`ws://localhost:7880`（`devkey` / `secret`）
2. HotGo：`cd ../server/server && air`（`:8000`）
3. 本客户端：

```bash
pnpm install
pnpm dev
```

浏览器打开 <http://127.0.0.1:5173>，输入房间名和昵称进房。

开发期 `/api` 由 Vite 代理到 HotGo，无需手配 CORS。

## 目录

- `src/views/JoinView.vue`：入会
- `src/views/RoomView.vue`：会议页与四个控制按钮
- `src/composables/useLiveKitRoom.ts`：唯一持有 LiveKit `Room`
- `src/api/conference.ts`：调用 `POST /api/conference/token/create`
