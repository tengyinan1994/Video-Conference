/** 是否运行在 Tauri 桌面壳内（网页模式为 false） */
export function isTauri(): boolean {
  return typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window
}

/** 是否 macOS：屏幕/系统音频采集能力与平台强相关，提示文案要分平台 */
export function isMacOS(): boolean {
  if (typeof navigator === 'undefined') return false
  const ua = navigator.userAgent || ''
  return /Macintosh|Mac OS X/i.test(ua)
}
