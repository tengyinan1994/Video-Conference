/** 是否运行在 Tauri 桌面壳内（网页模式为 false） */
export function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
}
