/** 预热/枚举用的设备描述（不依赖 DOM MediaDeviceInfo） */
export type MediaDeviceHint = {
  kind: string
  deviceId?: string
}

export type LocalTrackLike = { stop: () => void }

export type CreateLocalTracksOpts = { audio: boolean; video: boolean }

export type WarmUpMediaDeps = {
  createLocalTracks: (opts: CreateLocalTracksOpts) => Promise<LocalTrackLike[]>
  enumerateDevices: () => Promise<MediaDeviceHint[]>
}

/** 是否存在带真实 deviceId 的 videoinput（空 deviceId 视为占位，不可采集） */
export function hasRealVideoInput(devices: MediaDeviceHint[]): boolean {
  return devices.some((d) => d.kind === 'videoinput' && !!d.deviceId)
}

/**
 * 媒体权限预热顺序（Windows WebView2 无摄像头时请求 video 可能原生崩溃）：
 * 1) 只预热麦克风；
 * 2) 再枚举设备；
 * 3) 仅当存在带 deviceId 的 videoinput 时才可选预热视频，失败忽略。
 */
export async function warmUpMediaPermissionsWith(deps: WarmUpMediaDeps): Promise<void> {
  const audioTracks = await deps.createLocalTracks({ audio: true, video: false })
  audioTracks.forEach((t) => t.stop())

  const devices = await deps.enumerateDevices()
  if (!hasRealVideoInput(devices)) return

  try {
    const videoTracks = await deps.createLocalTracks({ audio: false, video: true })
    videoTracks.forEach((t) => t.stop())
  } catch {
    // 视频权限被拒或采集失败不阻断；无摄像头机绝不应走到这里
  }
}
