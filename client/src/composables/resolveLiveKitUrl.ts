/** 解析 LiveKit 信令地址时用到的运行环境（便于单测，避免依赖 location / Tauri） */
export type LiveKitUrlEnv = {
  isTauri: boolean
  /** 浏览器 location.protocol，如 `https:`；无 location 时省略 */
  protocol?: string
  /** 浏览器 location.host，如 `meet.example:17885`；无 location 时省略 */
  host?: string
  isDev?: boolean
}

/**
 * 解析 LiveKit 信令地址。
 * - Tauri 壳：始终用 token.serverUrl（webview 的 host 不是会议站点）
 * - 浏览器 https：同源 wss（nginx / Vite 反代 /rtc），避免混合内容被拦
 * - 浏览器 http + DEV：同源 ws（Vite 代理 /rtc）
 * - 浏览器 http 生产 / 无 location：原样返回 serverUrl
 */
export function resolveLiveKitUrlFromEnv(serverUrl: string, env: LiveKitUrlEnv): string {
  if (env.isTauri) return serverUrl
  if (env.protocol === undefined || env.host === undefined) return serverUrl
  if (env.isDev || env.protocol === 'https:') {
    const proto = env.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${env.host}`
  }
  return serverUrl
}
