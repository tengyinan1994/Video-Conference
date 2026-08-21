import { computed, onUnmounted, ref, shallowRef, watch } from 'vue'
import {
  ConnectionQuality,
  ConnectionState,
  createLocalTracks,
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
function isEgressParticipant(p: Participant): boolean {
  const identity = (p.identity || '').trim()
  const name = (p.name || '').trim()
  if (identity.startsWith('EG_') || name.startsWith('EG_')) return true
  const kind = (p as { kind?: string | number }).kind
  return kind === 'egress' || kind === 3
}

function isMediaTrackLive(track: { mediaStreamTrack?: MediaStreamTrack } | undefined | null): boolean {
  if (!track) return false
  const mst = track.mediaStreamTrack
  // 无 mediaStreamTrack 时按存在即有效；有则必须仍在 live
  if (!mst) return true
  return mst.readyState === 'live'
}

/** 摄像头/投屏 publication 是否仍有可播放画面（排除已 ended 的黑轨） */
function isVideoPublicationActive(
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
  isSpeaking: boolean
  isCameraEnabled: boolean
  isMicrophoneEnabled: boolean
  isScreenSharing: boolean
  connectionQuality: QualityLevel
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

function mediaErrorMessage(err: unknown): string {
  const name = err instanceof DOMException ? err.name : ''
  const message = err instanceof Error ? err.message : String(err)
  if (name === 'NotAllowedError' || message.includes('Permission')) {
    return '摄像头/麦克风权限被拒绝，请在浏览器设置中允许后重试'
  }
  if (name === 'NotReadableError' || message.includes('Device in use')) {
    return '设备被其他应用占用，请关闭占用后重试'
  }
  if (message.includes('secure') || message.includes('getUserMedia')) {
    return '当前页面不是安全上下文，请使用 localhost 或 https 访问'
  }
  if (message.includes('Failed to fetch') || message.includes('signal connection')) {
    return '无法连接 LiveKit 信令服务。请确认 livekit-server 已启动，或刷新后重试'
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

function parseRoleHost(metadata: string | undefined): boolean {
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
  const chatMessages = ref<ChatMessage[]>([])
  /** 本地主视图钉选（每人自己选，不广播）；为多人同时共享时切换主画面预留 */
  const focusedIdentity = ref<string | null>(null)
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
  const layoutMode = computed<LayoutMode>(() => (hasActiveVideo.value ? 'speaker' : 'avatar'))

  /** 主画面：优先本地钉选（须仍有画面）→ 投屏者 → 开摄像头的说话人 → 任一有画面的人 */
  const speakerParticipant = computed(() => {
    const list = participants.value
    if (!list.length) return null
    const withVideo = list.filter((p) => p.isCameraEnabled || p.isScreenSharing)
    if (focusedIdentity.value) {
      const pinned = withVideo.find((p) => p.identity === focusedIdentity.value)
      if (pinned) return pinned
    }
    const sharer = withVideo.find((p) => p.isScreenSharing)
    if (sharer) return sharer
    const active = withVideo.find((p) => p.identity === activeSpeakerId.value)
    if (active) return active
    const speaking = withVideo.find((p) => p.isSpeaking)
    if (speaking) return speaking
    return withVideo.find((p) => !p.isLocal) ?? withVideo[0] ?? null
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
    }
  })

  watch(
    participants,
    (list) => {
      if (!focusedIdentity.value) return
      const pinned = list.find((p) => p.identity === focusedIdentity.value)
      // 人已离开，或钉选对象已无画面：改钉到仍有画面的人
      if (!pinned || !(pinned.isCameraEnabled || pinned.isScreenSharing)) {
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
  }

  function rebuildParticipants() {
    const current = room.value
    if (!current) {
      participants.value = []
      screenSharing.value = false
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
  }

  function bindRoomEvents(r: Room) {
    const refresh = () => rebuildParticipants()

    r.on(RoomEvent.ConnectionStateChanged, (state: ConnectionState) => {
      if (state === ConnectionState.Connecting) status.value = 'connecting'
      else if (state === ConnectionState.Connected) {
        status.value = 'connected'
        errorMessage.value = ''
      } else if (state === ConnectionState.Reconnecting) status.value = 'reconnecting'
      else if (state === ConnectionState.Disconnected) status.value = 'disconnected'
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
    r.on(RoomEvent.Disconnected, () => {
      status.value = 'disconnected'
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

  async function connect(
    serverUrl: string,
    token: string,
    opts?: { enableMic?: boolean; enableCamera?: boolean },
  ) {
    await disconnect()
    status.value = 'connecting'
    errorMessage.value = ''
    qualityMap.clear()
    chatMessages.value = []
    activeSpeakerId.value = ''
    focusedIdentity.value = null

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
    qualityMap.clear()
    if (current) {
      try {
        await current.disconnect(true)
      } catch {
        // ignore
      }
    }
    if (status.value !== 'error') {
      status.value = 'idle'
    }
  }

  async function toggleMic() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isMicrophoneEnabled
    try {
      await local.setMicrophoneEnabled(next)
      micEnabled.value = local.isMicrophoneEnabled
      rebuildParticipants()
      if (local.isMicrophoneEnabled) void refreshDevices()
    } catch (err) {
      errorMessage.value = mediaErrorMessage(err)
      throw err
    }
  }

  async function toggleCamera() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isCameraEnabled
    try {
      // 摄像头与屏幕共享互斥：开摄像头前先停共享
      if (next && local.isScreenShareEnabled) {
        await local.setScreenShareEnabled(false)
      }
      await local.setCameraEnabled(next)
      cameraEnabled.value = local.isCameraEnabled
      screenSharing.value = local.isScreenShareEnabled
      rebuildParticipants()
      if (local.isCameraEnabled) void refreshDevices()
    } catch (err) {
      errorMessage.value = mediaErrorMessage(err)
      throw err
    }
  }

  async function toggleScreenShare() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isScreenShareEnabled
    const cameraWasOn = local.isCameraEnabled
    try {
      if (next) {
        // 摄像头与屏幕共享互斥：开共享前先关摄像头
        if (cameraWasOn) {
          await local.setCameraEnabled(false)
        }
        await local.setScreenShareEnabled(
          true,
          {
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
      errorMessage.value = mediaErrorMessage(err)
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
    toggleMic,
    toggleCamera,
    toggleScreenShare,
    sendChat,
    refreshDevices,
    switchMic,
    switchCamera,
    switchSpeaker,
    focusParticipant,
    clearFocus,
  }
}

/** 预热权限以便枚举到带 label 的设备列表（可选调用） */
export async function warmUpMediaPermissions() {
  const tracks = await createLocalTracks({ audio: true, video: true })
  tracks.forEach((t: LocalTrack) => t.stop())
}
