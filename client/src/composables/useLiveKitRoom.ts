import { computed, onUnmounted, ref, shallowRef } from 'vue'
import {
  ConnectionState,
  type LocalParticipant,
  type Participant,
  type RemoteParticipant,
  Room,
  RoomEvent,
  Track,
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

/** 仅保留挂载到 video/audio 元素所需能力，避开 LiveKit 私有字段的类型摩擦 */
export interface AttachableTrack {
  kind: string
  attach: (element: HTMLMediaElement) => HTMLMediaElement
  detach: (element?: HTMLMediaElement) => HTMLMediaElement[]
}

export interface MediaParticipant {
  identity: string
  name: string
  isLocal: boolean
  videoTrack?: AttachableTrack
  audioTrack?: AttachableTrack
  isSpeaking: boolean
  isCameraEnabled: boolean
  isMicrophoneEnabled: boolean
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
  return message || '媒体连接失败'
}

/** 开发环境经 Vite 同源代理 /rtc，避免内置预览拦直连 :7880 */
export function resolveLiveKitUrl(serverUrl: string): string {
  if (!import.meta.env.DEV || typeof location === 'undefined') {
    return serverUrl
  }
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}`
}

export function useLiveKitRoom() {
  const room = shallowRef<Room | null>(null)
  const status = ref<ConnectionStatus>('idle')
  const errorMessage = ref('')
  const participants = ref<MediaParticipant[]>([])
  const micEnabled = ref(true)
  const cameraEnabled = ref(true)
  const screenSharing = ref(false)

  const isConnected = computed(() => status.value === 'connected')

  function rebuildParticipants() {
    const current = room.value
    if (!current) {
      participants.value = []
      return
    }

    const list: MediaParticipant[] = []
    const push = (p: Participant, isLocal: boolean) => {
      const cam = p.getTrackPublication(Track.Source.Camera)
      const mic = p.getTrackPublication(Track.Source.Microphone)
      const screen = p.getTrackPublication(Track.Source.ScreenShare)
      const videoPub = screen?.track ? screen : cam
      list.push({
        identity: p.identity,
        name: p.name || p.identity,
        isLocal,
        videoTrack: videoPub?.track as AttachableTrack | undefined,
        audioTrack: mic?.track as AttachableTrack | undefined,
        isSpeaking: p.isSpeaking,
        isCameraEnabled: isLocal
          ? (p as LocalParticipant).isCameraEnabled
          : !!cam?.track && !cam.isMuted,
        isMicrophoneEnabled: isLocal
          ? (p as LocalParticipant).isMicrophoneEnabled
          : !!mic?.track && !mic.isMuted,
      })
    }

    push(current.localParticipant, true)
    current.remoteParticipants.forEach((p) => push(p, false))
    participants.value = list
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
    r.on(RoomEvent.ActiveSpeakersChanged, refresh)
    r.on(RoomEvent.LocalTrackPublished, refresh)
    r.on(RoomEvent.LocalTrackUnpublished, refresh)
    r.on(RoomEvent.TrackMuted, refresh)
    r.on(RoomEvent.TrackUnmuted, refresh)
    r.on(
      RoomEvent.TrackSubscribed,
      (_track: RemoteTrack, _pub: RemoteTrackPublication, _participant: RemoteParticipant) => {
        refresh()
      },
    )
    r.on(
      RoomEvent.TrackUnsubscribed,
      (_track: RemoteTrack, _pub: RemoteTrackPublication, _participant: RemoteParticipant) => {
        refresh()
      },
    )
    r.on(RoomEvent.Disconnected, () => {
      status.value = 'disconnected'
      refresh()
    })
  }

  async function connect(serverUrl: string, token: string) {
    await disconnect()
    status.value = 'connecting'
    errorMessage.value = ''

    const r = new Room({
      adaptiveStream: true,
      dynacast: true,
    })
    room.value = r
    bindRoomEvents(r)

    try {
      await r.connect(resolveLiveKitUrl(serverUrl), token)
      await r.localParticipant.setCameraEnabled(true)
      await r.localParticipant.setMicrophoneEnabled(true)
      micEnabled.value = r.localParticipant.isMicrophoneEnabled
      cameraEnabled.value = r.localParticipant.isCameraEnabled
      rebuildParticipants()
      // 本地轨发布后可能稍后才挂上 publication.track，再刷一次保证画面能绑上
      queueMicrotask(() => rebuildParticipants())
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
    await local.setMicrophoneEnabled(next)
    micEnabled.value = local.isMicrophoneEnabled
    rebuildParticipants()
  }

  async function toggleCamera() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isCameraEnabled
    await local.setCameraEnabled(next)
    cameraEnabled.value = local.isCameraEnabled
    rebuildParticipants()
  }

  async function toggleScreenShare() {
    const local = room.value?.localParticipant
    if (!local) return
    const next = !local.isScreenShareEnabled
    await local.setScreenShareEnabled(next)
    screenSharing.value = local.isScreenShareEnabled
    rebuildParticipants()
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
    connect,
    disconnect,
    toggleMic,
    toggleCamera,
    toggleScreenShare,
  }
}
