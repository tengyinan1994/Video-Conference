import { computed, nextTick, onUnmounted, ref, shallowRef, watch } from 'vue'
import {
  ConnectionQuality,
  ConnectionState,
  createLocalTracks,
  DisconnectReason,
  type LocalParticipant,
  type LocalTrack,
  type Participant,
  type RemoteParticipant,
  Room,
  RoomEvent,
  Track,
  VideoQuality,
  type RemoteTrack,
  type RemoteTrackPublication,
} from 'livekit-client'

export type ConnectionStatus =
  | 'idle'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'disconnected'
  | 'kicked'
  | 'ended'
  | 'error'

export type LayoutMode = 'avatar' | 'speaker'

export type QualityLevel = 'excellent' | 'good' | 'poor' | 'lost' | 'unknown'

export type AttachableTrack = {
  kind: string
  attach: (element: HTMLMediaElement) => HTMLMediaElement
  detach: (element?: HTMLMediaElement) => HTMLMediaElement[]
  mediaStreamTrack?: MediaStreamTrack
}

/** LiveKit Egress 会以 EG_xxx 虚拟参与者进房，不在成员列表展示 */
export function isEgressParticipant(p: Participant): boolean {
  const identity = (p.identity || '').trim()
  const name = (p.name || '').trim()
  if (identity.startsWith('EG_') || name.startsWith('EG_')) return true
  const kind = (p as { kind?: string | number }).kind
  return kind === 'egress' || kind === 3
}

export function isMediaTrackLive(track: { mediaStreamTrack?: MediaStreamTrack } | undefined | null): boolean {
  if (!track) return false
  const mst = track.mediaStreamTrack
  // 无 mediaStreamTrack 时按存在即有效；有则必须仍在 live
  if (!mst) return true
  return mst.readyState === 'live'
}

/** 摄像头/投屏 publication 是否仍有可播放画面（排除已 ended 的黑轨） */
export function isVideoPublicationActive(
  pub: { isMuted: boolean; track?: { mediaStreamTrack?: MediaStreamTrack } | undefined; isSubscribed?: boolean } | undefined,
  isLocal: boolean,
): boolean {
  if (!pub || pub.isMuted || !pub.track) return false
  if (!isLocal && pub.isSubscribed === false) return false
  return isMediaTrackLive(pub.track)
}

export interface MediaParticipant {
  identity: string
  name: string
  isLocal: boolean
  isHost: boolean
  /** 侧栏/宫格用人像：摄像头轨 */
  cameraTrack?: AttachableTrack
  /** 屏幕共享轨 */
  screenTrack?: AttachableTrack
  /** 默认展示轨：有投屏优先进投屏，否则摄像头 */
  videoTrack?: AttachableTrack
  audioTrack?: AttachableTrack
  /** 屏幕共享音频（标签页/系统声音），与麦克风分离 */
  screenAudioTrack?: AttachableTrack
  isSpeaking: boolean
  isCameraEnabled: boolean
  isMicrophoneEnabled: boolean
  isScreenSharing: boolean
  connectionQuality: QualityLevel
}

/** 远端需播放的音频轨：麦克风 + 屏幕共享音频（本地共享者不回放，避免自己听回音） */
export function collectRemoteAudioTracks(participants: MediaParticipant[]) {
  const items: { key: string; track: AttachableTrack }[] = []
  for (const p of participants) {
    if (p.isLocal) continue
    if (p.audioTrack) items.push({ key: `${p.identity}-mic`, track: p.audioTrack })
    if (p.screenAudioTrack) items.push({ key: `${p.identity}-screen-audio`, track: p.screenAudioTrack })
  }
  return items
}

export interface ChatMessage {
  id: string
  identity: string
  name: string
  text: string
  ts: number
  isLocal: boolean
}

export interface MediaDeviceOption {
  deviceId: string
  label: string
}

const CHAT_TOPIC = 'chat'

/** 用户在系统投屏选择器里点了取消（非真实权限故障） */
function isDisplayMediaCancelled(err: unknown): boolean {
  const name =
    err instanceof DOMException
      ? err.name
      : err && typeof err === 'object' && 'name' in err
        ? String((err as { name: unknown }).name)
        : ''
  const message = err instanceof Error ? err.message : String(err)
  if (name === 'AbortError') return true
  if (name === 'NotAllowedError') {
    // Chrome/Edge: "Permission denied by user"；部分环境仅 "Permission denied"
    return (
      message.includes('denied by user') ||
      message.includes('Permission denied') ||
      message.includes('NotAllowedError') ||
      message === '' ||
      /permission/i.test(message)
    )
  }
  return /Permission denied by user/i.test(message)
}

/** getDisplayMedia 是否可用：iOS Safari 及大部分 Android 浏览器不支持屏幕共享 */
function isScreenShareSupported(): boolean {
  if (typeof navigator === 'undefined') return false
  return typeof navigator.mediaDevices?.getDisplayMedia === 'function'
}

/**
 * Chrome/Edge 141+ 支持 restrictOwnAudio 约束：把「发起采集的这个标签页自己播放的声音」
 * 从系统音频里剔除。
 *
 * 为什么需要它：Windows 上「同时分享系统音频」抓的是默认播放设备的整段混音（WASAPI loopback），
 * 而其他参会人的声音正是由本页 <audio> 播放出来的，于是被一起抓走再发回房间，
 * 对方就听到自己的回声。加上这个约束后浏览器会把本页播放的声音减掉（内容声音仍保留），
 * 老版本浏览器会忽略这个未知约束，需要用提示兜底。
 */
export function supportsRestrictOwnAudio(): boolean {
  if (typeof navigator === 'undefined') return false
  if (typeof navigator.mediaDevices?.getSupportedConstraints !== 'function') return false
  const supported = navigator.mediaDevices.getSupportedConstraints() as Record<string, unknown>
  return supported.restrictOwnAudio === true
}

function connectionErrorReason(err: unknown): string {
  if (!err || typeof err !== 'object') return ''
  const reasonName = (err as { reasonName?: unknown }).reasonName
  return typeof reasonName === 'string' ? reasonName : ''
}

export function mediaErrorMessage(err: unknown): string {
  // 兼容不同错误形态：浏览器 DOMException / JS Error / LiveKit 自定义错误 / 普通对象 { name, message }
  let name = ''
  let message = ''
  if (err instanceof DOMException || err instanceof Error) {
    name = err.name || ''
    message = err.message || ''
  } else if (err && typeof err === 'object') {
    const obj = err as { name?: unknown; message?: unknown }
    name = typeof obj.name === 'string' ? obj.name : ''
    message = typeof obj.message === 'string' ? obj.message : ''
  } else if (err !== undefined && err !== null) {
    message = String(err)
  }
  const reason = connectionErrorReason(err)
  const status =
    err && typeof err === 'object' && 'status' in err
      ? Number((err as { status?: unknown }).status)
      : NaN
  // 摄像头/麦克风权限被浏览器拦截或拒绝。
  // 不同浏览器/版本抛出的文案差异很大（如 "Permission denied"、"permission denied"、
  // Firefox "denied permission"、独立对象 { name: 'NotAllowedError' } 等），统一归一化后大小写不敏感匹配。
  const lowerText = `${name} ${message}`.toLowerCase()
  if (
    name === 'NotAllowedError' ||
    lowerText.includes('notallowed') ||
    lowerText.includes('permission denied') ||
    lowerText.includes('permission')
  ) {
    return '未获得摄像头/麦克风权限，已被浏览器拦截。请点击浏览器地址栏的权限图标，允许使用摄像头和麦克风后，再点「取消静音」重试'
  }
  if (name === 'NotReadableError' || message.includes('Device in use')) {
    return '设备被其他应用占用，请关闭占用后重试'
  }
  if (name === 'DeviceUnsupportedError' || message.includes('getDisplayMedia not supported')) {
    // 移动端（iOS Safari / 大部分 Android 浏览器）没有 getDisplayMedia，无法屏幕共享
    return '当前设备或浏览器不支持屏幕共享，请使用电脑端浏览器'
  }
  if (message.includes('secure') || message.includes('getUserMedia')) {
    return '当前页面不是安全上下文，请使用 localhost 或 https 访问'
  }
  // LiveKit ConnectionError：区分票据无效 / 服务不可达，避免一律提示「server 未启动」
  if (reason === 'NotAllowed' || status === 401 || status === 403) {
    return '进房凭证无效或已失效，请返回后重新进入会议'
  }
  if (reason === 'Timeout' || message.includes('timed out')) {
    return '连接 LiveKit 超时。请检查网络后重试，或确认会议仍可加入'
  }
  if (
    reason === 'ServerUnreachable' ||
    reason === 'WebSocket' ||
    message.includes('Failed to fetch') ||
    message.includes('signal connection') ||
    message.includes('server was not reachable')
  ) {
    return '无法连接 LiveKit 信令服务。请确认已用 https 打开会议页并信任证书后刷新重试；若刚离开又进，请从分享链接重新进入'
  }
  if (
    message.includes('could not establish pc connection') ||
    message.includes('PC connection') ||
    message.includes('ICE failed')
  ) {
    return '媒体连接失败（WebRTC/ICE）。局域网开会请确认 LiveKit 的 node_ip 是宿主机局域网 IP，且 7881/7882 端口可从对方机器访问'
  }
  if (
    typeof window !== 'undefined' &&
    !window.isSecureContext &&
    (message.includes('getUserMedia') ||
      message.includes('getDisplayMedia') ||
      message.includes('Permission') ||
      name === 'NotAllowedError' ||
      name === 'SecurityError')
  ) {
    return '当前页面不是安全上下文（常见于 http://内网IP）。请使用 https://内网IP:5173 打开，或本机临时用 localhost'
  }
  return message || '媒体连接失败'
}

function mapQuality(q: ConnectionQuality): QualityLevel {
  switch (q) {
    case ConnectionQuality.Excellent:
      return 'excellent'
    case ConnectionQuality.Good:
      return 'good'
    case ConnectionQuality.Poor:
      return 'poor'
    case ConnectionQuality.Lost:
      return 'lost'
    default:
      return 'unknown'
  }
}

export function parseRoleHost(metadata: string | undefined): boolean {
  if (!metadata) return false
  try {
    const obj = JSON.parse(metadata) as { role?: string }
    return obj.role === 'host'
  } catch {
    return false
  }
}

/** LiveKit 信令地址：https 页必须同源 wss（nginx/Vite 反代 /rtc），避免混合内容被拦 */
export function resolveLiveKitUrl(serverUrl: string): string {
  if (typeof location === 'undefined') {
    return serverUrl
  }
  if (import.meta.env.DEV || location.protocol === 'https:') {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${location.host}`
  }
  return serverUrl
}


export function useLiveKitRoom() {
  const room = shallowRef<Room | null>(null)
  const status = ref<ConnectionStatus>('idle')
  const errorMessage = ref('')
  const participants = ref<MediaParticipant[]>([])
  const micEnabled = ref(false)
  const cameraEnabled = ref(false)
  const screenSharing = ref(false)
  /**
   * 本地投屏是否真的带上了一条屏幕音频轨。
   * Chrome 只在「标签页（且勾选同时分享标签页音频）」或「整个屏幕（且勾选同时分享系统音频）」时才给音轨；
   * 「窗口」共享以及没勾音频开关时 getDisplayMedia 根本不返回音轨，LiveKit 会静默地只发布视频。
   */
  const screenShareHasAudio = ref(false)
  /**
   * 本地投屏采集面（浏览器给的 displaySurface）：window / screen / browser。
   * 用来区分「窗口」和「整个屏幕」两种失败原因，给出不同提示。
   */
  const screenShareSurface = ref<'window' | 'screen' | 'browser' | ''>('')
  /**
   * 本次投屏是否已把「本页自己播放的声音」从系统音频里滤掉（Chrome 141+ 的 restrictOwnAudio）。
   * 为 false 且确实在共享系统音频时，其他人可能会听到自己的回声。
   */
  const screenShareOwnAudioFiltered = ref(false)
  /**
   * 本机投屏轨的发布时快照（分辨率 / 帧率），供「自己在投屏」的信息卡展示共享状态。
   * 只在 publish / rebuild 时从 getSettings() 取一次，不是实时仪表。
   */
  const screenShareStats = ref<{ width: number; height: number; frameRate: number } | null>(null)
  const chatMessages = ref<ChatMessage[]>([])
  /** 本地主视图钉选（每人自己选，不广播）；为多人同时共享时切换主画面预留 */
  const focusedIdentity = ref<string | null>(null)
  /** 钉选对象暂时无画面时，主视图回退到的人（保持稳定，不跟说话人跳） */
  const standbyIdentity = ref<string | null>(null)
  const activeSpeakerId = ref('')

  const audioInputs = ref<MediaDeviceOption[]>([])
  const videoInputs = ref<MediaDeviceOption[]>([])
  const audioOutputs = ref<MediaDeviceOption[]>([])
  const selectedMicId = ref('')
  const selectedCameraId = ref('')
  const selectedSpeakerId = ref('')
  const speakerSupported = ref(
    typeof HTMLMediaElement !== 'undefined' && 'setSinkId' in HTMLMediaElement.prototype,
  )

  const qualityMap = new Map<string, QualityLevel>()
  /** 是否已为枚举设备申请过媒体权限（默认关麦关摄像头时也要申请一次） */
  let devicePermissionWarmed = false
  /**
   * 终态原因（会议已结束 / 被移出）。一旦锁定就不再被后续
   * Disconnected / disconnect() 的收尾逻辑覆盖成 idle/disconnected，
   * 否则「会议已结束」的界面会被重置回可操作状态。
   */
  let terminalReason: 'ended' | 'kicked' | null = null

  const isConnected = computed(() => status.value === 'connected')

  /** 有人开摄像头或投屏时进入演讲者布局；否则头像墙 */
  const hasActiveVideo = computed(() =>
    participants.value.some((p) => p.isCameraEnabled || p.isScreenSharing),
  )
  /** 当前有画面（摄像头或投屏）的人数 */
  const activeVideoCount = computed(
    () => participants.value.filter((p) => p.isCameraEnabled || p.isScreenSharing).length,
  )
  /** 超过 1 人有画面时才显示右侧成员切换栏 */
  const showSpeakerSide = computed(() => activeVideoCount.value > 1)
  const anyoneScreenSharing = computed(() => participants.value.some((p) => p.isScreenSharing))
  /** 正在投屏但浏览器没给音轨：远端听不到你播放的内容声音（说话声不受影响） */
  const screenShareAudioMissing = computed(() => screenSharing.value && !screenShareHasAudio.value)
  const layoutMode = computed<LayoutMode>(() => (hasActiveVideo.value ? 'speaker' : 'avatar'))

  /** 该参与人是否有"当前仍可渲染"的视频轨（有媒体流且 live）。
   *  已 ended 的投屏/摄像头轨挂上 <video> 只会黑屏，优先排除，避免其霸占主视图。 */
  function hasRenderableVideo(
    p: { screenTrack?: AttachableTrack; cameraTrack?: AttachableTrack },
  ): boolean {
    const live = (t?: AttachableTrack | null) => !!t?.mediaStreamTrack && t.mediaStreamTrack.readyState === 'live'
    return live(p.screenTrack) || live(p.cameraTrack)
  }

  function pickVideoFallback(withVideo: MediaParticipant[]): MediaParticipant | null {
    // 优先在"仍有可渲染画面"的人里挑选；全部失效时退回原列表（主画面对失效轨已另行兜底为占位）
    const livePool = withVideo.filter((p) => hasRenderableVideo(p))
    const pool = livePool.length ? livePool : withVideo
    const standby = standbyIdentity.value
      ? pool.find((p) => p.identity === standbyIdentity.value)
      : undefined
    if (standby) return standby
    const sharer = pool.find((p) => p.isScreenSharing)
    if (sharer) return sharer
    const active = pool.find((p) => p.identity === activeSpeakerId.value)
    if (active) return active
    const speaking = pool.find((p) => p.isSpeaking)
    if (speaking) return speaking
    return pool.find((p) => !p.isLocal) ?? pool[0] ?? null
  }

  /** 主画面：优先本地钉选（须仍有画面）→ 稳定回退 → 投屏者 → 开摄像头的说话人 → 任一有画面的人 */
  const speakerParticipant = computed(() => {
    const list = participants.value
    if (!list.length) return null
    const withVideo = list.filter((p) => p.isCameraEnabled || p.isScreenSharing)
    if (focusedIdentity.value) {
      const pinned = withVideo.find((p) => p.identity === focusedIdentity.value)
      if (pinned) return pinned
    }
    return pickVideoFallback(withVideo)
  })

  watch(speakerParticipant, (p) => {
    if (p && focusedIdentity.value && p.identity === focusedIdentity.value) {
      standbyIdentity.value = null
      return
    }
    standbyIdentity.value = p?.identity ?? null
  })

  watch(hasActiveVideo, (on, wasOn) => {
    if (on && !wasOn) {
      // 刚有人出画面：未钉选时自动钉到投屏者或第一个开摄像头的人
      if (!focusedIdentity.value) {
        const sharer = participants.value.find((p) => p.isScreenSharing)
        const cam = participants.value.find((p) => p.isCameraEnabled)
        focusedIdentity.value = (sharer ?? cam)?.identity ?? null
      }
    } else if (!on) {
      focusedIdentity.value = null
      standbyIdentity.value = null
    }
  })

  watch(
    participants,
    (list) => {
      if (!focusedIdentity.value) return
      const pinned = list.find((p) => p.identity === focusedIdentity.value)
      // 仅在钉选对象离会时改钉。关摄像头是临时无画面：主视图先回退到其他人，
      // 但保留钉选，对方重新开摄像头后应回到主视图（名字与画面一致）。
      if (!pinned) {
        const next = list.find((p) => p.isCameraEnabled || p.isScreenSharing)
        focusedIdentity.value = next?.identity ?? null
      }
    },
    { deep: true },
  )

  function focusParticipant(identity: string) {
    if (!identity) return
    if (!participants.value.some((p) => p.identity === identity)) return
    focusedIdentity.value = identity
  }

  function clearFocus() {
    focusedIdentity.value = null
    standbyIdentity.value = null
  }

  function rebuildParticipants() {
    const current = room.value
    if (!current) {
      participants.value = []
      screenSharing.value = false
      screenShareHasAudio.value = false
      screenShareSurface.value = ''
      screenShareOwnAudioFiltered.value = false
      screenShareStats.value = null
      micEnabled.value = false
      cameraEnabled.value = false
      return
    }

    const list: MediaParticipant[] = []
    const push = (p: Participant, isLocal: boolean) => {
      if (isEgressParticipant(p)) return
      const cam = p.getTrackPublication(Track.Source.Camera)
      const mic = p.getTrackPublication(Track.Source.Microphone)
      const screen = p.getTrackPublication(Track.Source.ScreenShare)
      const screenAudio = p.getTrackPublication(Track.Source.ScreenShareAudio)
      const cameraOn = isVideoPublicationActive(cam, isLocal)
      const screenOn = isVideoPublicationActive(screen, isLocal)
      const screenTrack = screenOn ? (screen?.track as AttachableTrack | undefined) : undefined
      const cameraTrack = cameraOn ? (cam?.track as AttachableTrack | undefined) : undefined
      list.push({
        identity: p.identity,
        name: p.name || p.identity,
        isLocal,
        isHost: parseRoleHost(p.metadata),
        cameraTrack,
        screenTrack,
        videoTrack: screenTrack ?? cameraTrack,
        audioTrack: mic?.track as AttachableTrack | undefined,
        screenAudioTrack: screenAudio?.track as AttachableTrack | undefined,
        isSpeaking: p.isSpeaking,
        isCameraEnabled: cameraOn,
        isMicrophoneEnabled: isLocal
          ? (p as LocalParticipant).isMicrophoneEnabled
          : !!mic?.track && !mic.isMuted,
        isScreenSharing: screenOn,
        connectionQuality: qualityMap.get(p.identity) ?? 'unknown',
      })
    }

    push(current.localParticipant, true)
    current.remoteParticipants.forEach((p) => push(p, false))
    participants.value = list
    const local = current.localParticipant
    // 服务端全员静音等远程 mute 只触发 TrackMuted，需同步工具栏按钮状态
    micEnabled.value = local.isMicrophoneEnabled
    cameraEnabled.value = local.isCameraEnabled
    screenSharing.value = local.isScreenShareEnabled
    // 浏览器是否真的把屏幕音频交给我们了（决定要不要提示用户「对方听不到视频声音」）
    const localScreenAudio = local.getTrackPublication(Track.Source.ScreenShareAudio)
    screenShareHasAudio.value = !!(
      localScreenAudio &&
      !localScreenAudio.isMuted &&
      isMediaTrackLive(localScreenAudio.track)
    )
    // 采集面 + 共享状态快照：Chrome 只在 video track settings 里给 displaySurface，顺带取分辨率/帧率
    const localScreen = local.getTrackPublication(Track.Source.ScreenShare)
    const screenSettings = localScreen?.track?.mediaStreamTrack?.getSettings() as
      | { displaySurface?: string; width?: number; height?: number; frameRate?: number }
      | undefined
    const surface = screenSettings?.displaySurface
    screenShareSurface.value =
      surface === 'window' || surface === 'screen' || surface === 'browser' ? surface : ''
    screenShareStats.value =
      screenShareSurface.value && screenSettings?.width && screenSettings?.height
        ? {
            width: screenSettings.width,
            height: screenSettings.height,
            frameRate: Math.round(screenSettings.frameRate ?? 0),
          }
        : null
    // 回声防护是否生效：以浏览器回报的 settings 为准，未回报时按「支持该约束」推断
    const ownAudio = (localScreenAudio?.track?.mediaStreamTrack?.getSettings() as
      | { restrictOwnAudio?: boolean }
      | undefined)?.restrictOwnAudio
    screenShareOwnAudioFiltered.value =
      screenShareHasAudio.value && (ownAudio === true || supportsRestrictOwnAudio())
  }

  function bindRoomEvents(r: Room) {
    const refresh = () => rebuildParticipants()

    r.on(RoomEvent.ConnectionStateChanged, (state: ConnectionState) => {
      if (state === ConnectionState.Connecting) status.value = 'connecting'
      else if (state === ConnectionState.Connected) {
        status.value = 'connected'
        errorMessage.value = ''
      } else if (state === ConnectionState.Reconnecting) status.value = 'reconnecting'
      // Disconnected：交给 RoomEvent.Disconnected，以便区分踢出 / 会议结束等原因
    })

    r.on(RoomEvent.ParticipantConnected, refresh)
    r.on(RoomEvent.ParticipantDisconnected, refresh)
    r.on(RoomEvent.ParticipantMetadataChanged, refresh)
    r.on(RoomEvent.ActiveSpeakersChanged, (speakers: Participant[]) => {
      activeSpeakerId.value = speakers[0]?.identity ?? ''
      refresh()
    })
    r.on(RoomEvent.LocalTrackPublished, (pub) => {
      refresh()
      // 浏览器原生「停止分享」会结束轨道；LiveKit 会 unpublish，这里再兜底刷新 UI
      if (pub.source === Track.Source.ScreenShare && pub.track) {
        const media = pub.track.mediaStreamTrack
        const onEnded = () => {
          screenSharing.value = false
          // 确保彻底下架，避免远端一直订到黑屏轨
          void r.localParticipant.setScreenShareEnabled(false).finally(() => {
            rebuildParticipants()
          })
        }
        media?.addEventListener('ended', onEnded, { once: true })
      }
    })
    r.on(RoomEvent.LocalTrackUnpublished, () => {
      refresh()
      const local = r.localParticipant
      screenSharing.value = local.isScreenShareEnabled
    })
    r.on(RoomEvent.TrackMuted, refresh)
    r.on(RoomEvent.TrackUnmuted, refresh)
    r.on(RoomEvent.TrackPublished, refresh)
    r.on(RoomEvent.TrackUnpublished, refresh)
    r.on(RoomEvent.AudioPlaybackStatusChanged, () => {
      // 自动播放被拦截或投屏后被暂停时尝试恢复
      if (!r.canPlaybackAudio) return
      void resumeAudioPlayback()
    })
    r.on(
      RoomEvent.TrackSubscribed,
      (track: RemoteTrack, pub: RemoteTrackPublication, _participant: RemoteParticipant) => {
        // 投屏轨强制要最高订阅质量，避免 adaptiveStream 卡在糊档
        if (pub.source === Track.Source.ScreenShare) {
          pub.setVideoQuality(VideoQuality.HIGH)
          track.mediaStreamTrack?.addEventListener(
            'ended',
            () => {
              rebuildParticipants()
            },
            { once: true },
          )
        }
        refresh()
        void applySpeakerOutput()
      },
    )
    r.on(
      RoomEvent.TrackUnsubscribed,
      (_track: RemoteTrack, _pub: RemoteTrackPublication, _participant: RemoteParticipant) => {
        refresh()
      },
    )
    r.on(RoomEvent.ConnectionQualityChanged, (quality: ConnectionQuality, participant?: Participant) => {
      const id = participant?.identity ?? r.localParticipant.identity
      qualityMap.set(id, mapQuality(quality))
      refresh()
    })
    r.on(
      RoomEvent.DataReceived,
      (payload: Uint8Array, participant?: RemoteParticipant, _kind?: unknown, topic?: string) => {
        if (topic && topic !== CHAT_TOPIC) return
        try {
          const raw = new TextDecoder().decode(payload)
          const data = JSON.parse(raw) as { type?: string; text?: string; ts?: number }
          if (data.type !== 'chat' || !data.text) return
          const identity = participant?.identity ?? 'unknown'
          chatMessages.value = [
            ...chatMessages.value,
            {
              id: `${identity}-${data.ts ?? Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
              identity,
              name: participant?.name || identity,
              text: data.text,
              ts: data.ts ?? Date.now(),
              isLocal: false,
            },
          ]
        } catch {
          // ignore malformed
        }
      },
    )
    r.on(RoomEvent.Disconnected, (reason?: DisconnectReason) => {
      // 已锁定终态（如轮询先行判定会议已结束）时忽略后到的断开原因，避免覆盖提示
      if (terminalReason) {
        refresh()
        return
      }
      if (reason === DisconnectReason.PARTICIPANT_REMOVED) {
        void markTerminal('kicked')
      } else if (
        reason === DisconnectReason.ROOM_DELETED ||
        reason === DisconnectReason.ROOM_CLOSED
      ) {
        void markTerminal('ended')
      } else if (reason === DisconnectReason.DUPLICATE_IDENTITY) {
        status.value = 'disconnected'
        errorMessage.value = '相同账号已在其他设备进入会议，当前连接已断开'
      } else {
        status.value = 'disconnected'
      }
      refresh()
    })
  }

  async function ensureDevicePermission() {
    if (devicePermissionWarmed) return
    const local = room.value?.localParticipant
    // 已开麦/摄像头说明权限在，无需再预热
    if (local?.isMicrophoneEnabled || local?.isCameraEnabled) {
      devicePermissionWarmed = true
      return
    }
    try {
      await warmUpMediaPermissions()
      devicePermissionWarmed = true
    } catch {
      // 部分环境视频权限会拦整次；至少抢麦克风权限以便枚举
      try {
        const tracks = await createLocalTracks({ audio: true, video: false })
        tracks.forEach((t: LocalTrack) => t.stop())
        devicePermissionWarmed = true
      } catch {
        // 用户拒绝或非安全上下文（如 http://内网IP）时列表可能仍为空
      }
    }
  }

  async function refreshDevices() {
    try {
      // 默认关麦关摄像头进房时从未要过权限，Chrome 会返回空列表或空 label
      await ensureDevicePermission()

      // 第二参数 false：权限已在 ensure 里处理；避免非安全上下文下 requestPermissions 卡住
      const devices = await Room.getLocalDevices(undefined, false)
      audioInputs.value = devices
        .filter((d) => d.kind === 'audioinput' && d.deviceId)
        .map((d) => ({
          deviceId: d.deviceId,
          label: d.label || `麦克风 ${d.deviceId.slice(0, 6)}`,
        }))
      videoInputs.value = devices
        .filter((d) => d.kind === 'videoinput' && d.deviceId)
        .map((d) => ({
          deviceId: d.deviceId,
          label: d.label || `摄像头 ${d.deviceId.slice(0, 6)}`,
        }))
      audioOutputs.value = devices
        .filter((d) => d.kind === 'audiooutput' && d.deviceId)
        .map((d) => ({
          deviceId: d.deviceId,
          label: d.label || `扬声器 ${d.deviceId.slice(0, 6)}`,
        }))

      const r = room.value
      const pick = (active: string | undefined, list: MediaDeviceOption[]) => {
        if (active && list.some((d) => d.deviceId === active)) return active
        return list[0]?.deviceId || ''
      }
      if (r) {
        selectedMicId.value = pick(r.getActiveDevice('audioinput'), audioInputs.value)
        selectedCameraId.value = pick(r.getActiveDevice('videoinput'), videoInputs.value)
        selectedSpeakerId.value = pick(r.getActiveDevice('audiooutput'), audioOutputs.value)
      } else {
        selectedMicId.value = audioInputs.value[0]?.deviceId || ''
        selectedCameraId.value = videoInputs.value[0]?.deviceId || ''
        selectedSpeakerId.value = audioOutputs.value[0]?.deviceId || ''
      }
    } catch {
      // 权限未授予时列表可能为空
    }
  }

  function bindDeviceChangeListener() {
    if (typeof navigator === 'undefined' || !navigator.mediaDevices?.addEventListener) return
    const onChange = () => {
      void refreshDevices()
    }
    navigator.mediaDevices.addEventListener('devicechange', onChange)
    onUnmounted(() => {
      navigator.mediaDevices.removeEventListener('devicechange', onChange)
    })
  }

  bindDeviceChangeListener()

  async function applySpeakerOutput() {
    const deviceId = selectedSpeakerId.value
    if (!deviceId || !speakerSupported.value) return
    const audios = document.querySelectorAll<HTMLAudioElement>('audio')
    for (const el of audios) {
      try {
        // setSinkId 非所有浏览器支持
        await (el as HTMLAudioElement & { setSinkId?: (id: string) => Promise<void> }).setSinkId?.(
          deviceId,
        )
      } catch {
        // ignore
      }
    }
  }

  /** 恢复远端音频播放（投屏/自动播放策略可能暂停 audio 元素） */
  async function resumeAudioPlayback() {
    const r = room.value
    if (r) {
      try {
        await r.startAudio()
      } catch {
        // 无用户手势时浏览器可能拒绝，忽略
      }
    }
    const audios = document.querySelectorAll<HTMLAudioElement>('audio')
    for (const el of audios) {
      try {
        if (el.paused) await el.play()
      } catch {
        // ignore
      }
    }
    await applySpeakerOutput()
  }

  async function connect(
    serverUrl: string,
    token: string,
    opts?: { enableMic?: boolean; enableCamera?: boolean },
  ) {
    await disconnect()
    terminalReason = null
    status.value = 'connecting'
    errorMessage.value = ''
    qualityMap.clear()
    chatMessages.value = []
    activeSpeakerId.value = ''
    focusedIdentity.value = null
    standbyIdentity.value = null

    const wantMic = !!opts?.enableMic
    const wantCamera = !!opts?.enableCamera
    devicePermissionWarmed = false

    const r = new Room({
      // 2K/Retina 上按更高像素密度要流，投屏文字更清晰
      adaptiveStream: { pixelDensity: 2 },
      dynacast: true,
      publishDefaults: {
        // 投屏目标：2K@60，弱网时优先保分辨率（文字更清晰）
        screenShareEncoding: {
          maxBitrate: 12_000_000,
          maxFramerate: 60,
        },
        degradationPreference: 'maintain-resolution',
      },
    })
    room.value = r
    bindRoomEvents(r)

    try {
      await r.connect(resolveLiveKitUrl(serverUrl), token)
      // 默认关麦关摄像头；仅在进房前显式选择开启时才请求设备
      await r.localParticipant.setMicrophoneEnabled(wantMic)
      await r.localParticipant.setCameraEnabled(wantCamera)
      micEnabled.value = r.localParticipant.isMicrophoneEnabled
      cameraEnabled.value = r.localParticipant.isCameraEnabled
      qualityMap.set(r.localParticipant.identity, mapQuality(r.localParticipant.connectionQuality))
      rebuildParticipants()
      queueMicrotask(() => rebuildParticipants())
      // 设备枚举失败不阻断进房
      void refreshDevices()
      status.value = 'connected'
    } catch (err) {
      status.value = 'error'
      errorMessage.value = mediaErrorMessage(err)
      await disconnect()
      throw err
    }
  }

  async function disconnect() {
    const current = room.value
    room.value = null
    participants.value = []
    screenSharing.value = false
    focusedIdentity.value = null
    standbyIdentity.value = null
    qualityMap.clear()
    if (current) {
      // stopTracks=true：立即停止本地投屏/摄像头/麦克风采集，浏览器共享指示灯随之熄灭
      try {
        await current.disconnect(true)
      } catch {
        // ignore
      }
    }
    // 终态（会议已结束/被移出）不可被收尾逻辑重置，否则界面会回到可继续操作的状态
    if (!terminalReason && status.value !== 'error') {
      status.value = 'idle'
    }
  }

  /**
   * 锁定「会议已结束 / 被移出」终态并立即断开连接。
   * LiveKit 的 ROOM_DELETED/PARTICIPANT_REMOVED 事件，以及客户端的会议状态轮询兜底，
   * 都走这里，保证媒体采集被立刻释放、状态不会被后续 disconnect() 覆盖。
   */
  async function markTerminal(reason: 'ended' | 'kicked') {
    if (terminalReason) return
    terminalReason = reason
    status.value = reason
    errorMessage.value = reason === 'kicked' ? '你已被主持人移出会议' : '会议已结束'
    await disconnect()
    status.value = reason
  }

  async function toggleMic() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isMicrophoneEnabled
    await local.setMicrophoneEnabled(next)
    micEnabled.value = local.isMicrophoneEnabled
    rebuildParticipants()
    if (local.isMicrophoneEnabled) void refreshDevices()
  }

  async function toggleCamera() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isCameraEnabled
    // 摄像头与屏幕共享互斥：开摄像头前先停共享
    if (next && local.isScreenShareEnabled) {
      await local.setScreenShareEnabled(false)
    }
    await local.setCameraEnabled(next)
    cameraEnabled.value = local.isCameraEnabled
    screenSharing.value = local.isScreenShareEnabled
    rebuildParticipants()
    if (local.isCameraEnabled) void refreshDevices()
  }

  async function toggleScreenShare() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isScreenShareEnabled
    const cameraWasOn = local.isCameraEnabled
    // 移动端没有 getDisplayMedia：直接给出明确提示，避免 LiveKit 抛原始错误
    if (next && !isScreenShareSupported()) {
      throw new Error('当前设备或浏览器不支持屏幕共享，请使用电脑端浏览器')
    }
    try {
      if (next) {
        // 摄像头与屏幕共享互斥：开共享前先关摄像头
        if (cameraWasOn) {
          await local.setCameraEnabled(false)
        }
        await local.setScreenShareEnabled(
          true,
          {
            audio: {
              // 共享的是视频/标签页声音，不要当成人声做降噪，否则内容音频会被吃掉
              echoCancellation: false,
              noiseSuppression: false,
              autoGainControl: false,
              // Chrome 141+：把本页自己播放的声音（其他参会人的人声）从系统音频里减掉，否则对方听到回声
              ...(supportsRestrictOwnAudio() ? { restrictOwnAudio: true } : {}),
            },
            systemAudio: 'include',
            contentHint: 'detail',
            resolution: {
              width: 2560,
              height: 1440,
              frameRate: 60,
            },
          },
          {
            simulcast: false,
            screenShareEncoding: {
              maxBitrate: 12_000_000,
              maxFramerate: 60,
            },
            degradationPreference: 'maintain-resolution',
          },
        )
      } else {
        await local.setScreenShareEnabled(false)
      }
      screenSharing.value = local.isScreenShareEnabled
      cameraEnabled.value = local.isCameraEnabled
      rebuildParticipants()
      // 开启投屏后布局切到演讲者视图，浏览器也可能暂停已有 audio，强制恢复远端声音
      if (next && local.isScreenShareEnabled) {
        await nextTick()
        await resumeAudioPlayback()
      }
    } catch (err) {
      // 用户取消投屏选择器：不提示错误，并恢复此前关闭的摄像头
      if (next && isDisplayMediaCancelled(err)) {
        if (cameraWasOn && !local.isCameraEnabled) {
          try {
            await local.setCameraEnabled(true)
          } catch {
            // 恢复失败时保持现状即可
          }
        }
        screenSharing.value = local.isScreenShareEnabled
        cameraEnabled.value = local.isCameraEnabled
        rebuildParticipants()
        return
      }
      throw err
    }
  }

  async function sendChat(text: string) {
    const r = room.value
    if (!r || status.value !== 'connected') return
    const trimmed = text.trim()
    if (!trimmed) return
    const ts = Date.now()
    const payload = new TextEncoder().encode(JSON.stringify({ type: 'chat', text: trimmed, ts }))
    await r.localParticipant.publishData(payload, { reliable: true, topic: CHAT_TOPIC })
    chatMessages.value = [
      ...chatMessages.value,
      {
        id: `local-${ts}-${Math.random().toString(36).slice(2, 7)}`,
        identity: r.localParticipant.identity,
        name: r.localParticipant.name || r.localParticipant.identity,
        text: trimmed,
        ts,
        isLocal: true,
      },
    ]
  }

  async function switchMic(deviceId: string) {
    const r = room.value
    if (!r || !deviceId) return
    await r.switchActiveDevice('audioinput', deviceId)
    selectedMicId.value = deviceId
  }

  async function switchCamera(deviceId: string) {
    const r = room.value
    if (!r || !deviceId) return
    await r.switchActiveDevice('videoinput', deviceId)
    selectedCameraId.value = deviceId
  }

  async function switchSpeaker(deviceId: string) {
    if (!deviceId) return
    selectedSpeakerId.value = deviceId
    const r = room.value
    if (r) {
      try {
        await r.switchActiveDevice('audiooutput', deviceId)
      } catch {
        // 部分浏览器 Room.switchActiveDevice 对 audiooutput 支持有限，回退 setSinkId
      }
    }
    await applySpeakerOutput()
  }

  onUnmounted(() => {
    void disconnect()
  })

  return {
    room,
    status,
    errorMessage,
    participants,
    micEnabled,
    cameraEnabled,
    screenSharing,
    screenShareHasAudio,
    screenShareSurface,
    screenShareOwnAudioFiltered,
    screenShareAudioMissing,
    screenShareStats,
    isConnected,
    chatMessages,
    layoutMode,
    hasActiveVideo,
    activeVideoCount,
    showSpeakerSide,
    anyoneScreenSharing,
    focusedIdentity,
    activeSpeakerId,
    speakerParticipant,
    audioInputs,
    videoInputs,
    audioOutputs,
    selectedMicId,
    selectedCameraId,
    selectedSpeakerId,
    speakerSupported,
    connect,
    disconnect,
    markTerminal,
    toggleMic,
    toggleCamera,
    toggleScreenShare,
    sendChat,
    refreshDevices,
    switchMic,
    switchCamera,
    switchSpeaker,
    resumeAudioPlayback,
    focusParticipant,
    clearFocus,
  }
}

/** 预热权限以便枚举到带 label 的设备列表（可选调用） */
export async function warmUpMediaPermissions() {
  const tracks = await createLocalTracks({ audio: true, video: true })
  tracks.forEach((t: LocalTrack) => t.stop())
}
