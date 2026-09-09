<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import {
  ConnectionState,
  Room,
  RoomEvent,
  Track,
  VideoQuality,
  type LocalParticipant,
  type Participant,
  type RemoteParticipant,
  type RemoteTrack,
  type RemoteTrackPublication,
} from 'livekit-client'
import MediaTrack from '@/components/MediaTrack.vue'
import {
  isEgressParticipant,
  isVideoPublicationActive,
  parseRoleHost,
  type AttachableTrack,
  type MediaParticipant,
} from '@/composables/useLiveKitRoom'

/**
 * LiveKit Egress 录制模板页。
 * egress 会以 custom_base_url 打开本页，并自动追加 ?url=<livekit>&token=<recorder-token>&layout=<layout>。
 * 页面只订阅、不发布（recorder 为 subscribe-only），并渲染「会议舞台 + 参会者状态」，
 * 视图 ready 后无条件打印 START_RECORDING，从而在无人说话 / 无投屏 / 无摄像头时也能录制。
 */

const room = shallowRef<Room | null>(null)
const participants = ref<MediaParticipant[]>([])
const status = ref<'idle' | 'connecting' | 'connected' | 'error'>('idle')
const errorMessage = ref('')
const roomName = ref('')

let connected = false
let endSignalled = false

/** egress 注入的 url 可能是 ws(s):// 或 http(s)://，统一为 ws(s):// */
function normalizeWsUrl(u: string): string {
  const s = u.trim()
  if (s.startsWith('https://')) return `wss://${s.slice('https://'.length)}`
  if (s.startsWith('http://')) return `ws://${s.slice('http://'.length)}`
  return s
}

function signalStart() {
  console.log('START_RECORDING')
}

function signalEnd() {
  // 页面被卸载 / 房间断开时调用，作为 egress 兜底的结束信号
  if (endSignalled) return
  endSignalled = true
  console.log('END_RECORDING')
}

function pushParticipant(list: MediaParticipant[], p: Participant, isLocal: boolean) {
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
    connectionQuality: 'good',
  })
}

function rebuild() {
  const r = room.value
  if (!r) {
    participants.value = []
    return
  }
  const list: MediaParticipant[] = []
  pushParticipant(list, r.localParticipant, true)
  r.remoteParticipants.forEach((p) => pushParticipant(list, p, false))
  participants.value = list
}

// —— 复用会议室舞台布局（订阅只读渲染，不含任何交互控件）——
/** 有人开摄像头或投屏时进入演讲者布局；否则头像墙 */
const hasActiveVideo = computed(() =>
  participants.value.some((p) => p.isCameraEnabled || p.isScreenSharing),
)
/** 当前有画面（摄像头或投屏）的人数 */
const activeVideoCount = computed(
  () => participants.value.filter((p) => p.isCameraEnabled || p.isScreenSharing).length,
)
const layoutMode = computed<'speaker' | 'avatar'>(() => (hasActiveVideo.value ? 'speaker' : 'avatar'))
/** 超过 1 人有画面时才显示右侧成员小窗 */
const showSpeakerSide = computed(() => activeVideoCount.value > 1)
const avatarParticipants = computed(() => participants.value)
const sideParticipants = computed(() =>
  participants.value.filter((p) => p.isCameraEnabled || p.isScreenSharing),
)

/** 该轨当前是否可渲染：必须有媒体流且状态为 live，避免挂上已 ended 的轨变成黑屏 */
function canRenderTrack(track?: AttachableTrack | null): boolean {
  if (!track) return false
  const mst = track.mediaStreamTrack
  return !!mst && mst.readyState === 'live'
}

/** 取参与人当前可渲染的最佳视频轨：投屏轨若已失效则回退到摄像头轨 */
function pickVideoTrack(
  p?: MediaParticipant | null,
): { track?: AttachableTrack; isScreen: boolean } {
  if (!p) return { track: undefined, isScreen: false }
  if (canRenderTrack(p.screenTrack)) return { track: p.screenTrack, isScreen: true }
  if (canRenderTrack(p.cameraTrack)) return { track: p.cameraTrack, isScreen: false }
  return { track: undefined, isScreen: false }
}

function videoFor(p: MediaParticipant) {
  return pickVideoTrack(p)
}

/** 主讲：优先投屏 → 正在说话的人 → 任一有画面的人 */
const speakerParticipant = computed(() => {
  const withVideo = participants.value.filter((p) => p.isCameraEnabled || p.isScreenSharing)
  if (!withVideo.length) return null
  const sharer = withVideo.find((p) => p.isScreenSharing)
  if (sharer) return sharer
  const speaking = withVideo.find((p) => p.isSpeaking)
  if (speaking) return speaking
  return withVideo.find((p) => !p.isLocal) ?? withVideo[0] ?? null
})

const mainVideo = computed(() => {
  const p = speakerParticipant.value
  if (!p) return null
  const best = pickVideoTrack(p)
  return { participant: p, track: best.track, isScreen: best.isScreen }
})
const mainStageTrack = computed(() => mainVideo.value?.track)
const mainIsScreen = computed(() => !!mainVideo.value?.isScreen)

function bindRoomEvents(r: Room) {
  const refresh = () => rebuild()
  r.on(RoomEvent.ConnectionStateChanged, (state: ConnectionState) => {
    if (state === ConnectionState.Connecting) status.value = 'connecting'
    else if (state === ConnectionState.Connected) status.value = 'connected'
    else if (state === ConnectionState.Reconnecting) status.value = 'connecting'
  })
  r.on(RoomEvent.ParticipantConnected, refresh)
  r.on(RoomEvent.ParticipantDisconnected, refresh)
  r.on(RoomEvent.ParticipantMetadataChanged, refresh)
  r.on(RoomEvent.ActiveSpeakersChanged, refresh)
  r.on(RoomEvent.TrackMuted, refresh)
  r.on(RoomEvent.TrackUnmuted, refresh)
  r.on(RoomEvent.TrackPublished, refresh)
  r.on(RoomEvent.TrackUnpublished, refresh)
  r.on(
    RoomEvent.TrackSubscribed,
    (_track: RemoteTrack, pub: RemoteTrackPublication, _p: RemoteParticipant) => {
      // 投屏轨强制最高订阅质量，避免降档糊屏
      if (pub.source === Track.Source.ScreenShare) {
        pub.setVideoQuality(VideoQuality.HIGH)
      }
      refresh()
    },
  )
  r.on(RoomEvent.TrackUnsubscribed, refresh)
  r.on(RoomEvent.Disconnected, () => {
    status.value = 'idle'
    if (connected) signalEnd()
    connected = false
  })
}

async function connect(r: Room, serverUrl: string, token: string) {
  room.value = r
  if (r.name) roomName.value = r.name
  bindRoomEvents(r)
  try {
    await r.connect(normalizeWsUrl(serverUrl), token)
    connected = true
    roomName.value = r.name || roomName.value
    rebuild()
    // 视图就绪后立即无条件发 start，不依赖房间是否有轨道
    await nextTick()
    signalStart()
  } catch {
    status.value = 'error'
    errorMessage.value = '录制模板无法连接房间'
  }
}

onMounted(() => {
  const params = new URLSearchParams(window.location.search)
  const serverUrl = params.get('url') || ''
  const token = params.get('token') || ''
  if (!serverUrl || !token) {
    status.value = 'error'
    errorMessage.value = '缺少录制连接参数（url / token）'
    return
  }
  status.value = 'connecting'
  // 录制模板要稳定且尽可能按订阅质量抓取，关闭自适应/动态编码
  const r = new Room({ adaptiveStream: false, dynacast: false })
  void connect(r, serverUrl, token)
})

onBeforeUnmount(() => {
  if (connected) signalEnd()
  connected = false
  const r = room.value
  room.value = null
  if (r) {
    try {
      void r.disconnect(true)
    } catch {
      // ignore
    }
  }
})
</script>

<template>
  <div class="egress">
    <div class="egress-header">
      <div class="egress-roomslug">
        <span class="room-label">
          {{ roomName || '会议室' }}
        </span>
        <span class="room-sub">会议录制</span>
      </div>
      <span v-if="status === 'connecting'" class="state">正在进入房间…</span>
      <span v-else-if="status === 'connected'" class="state ok">录制中</span>
      <span v-else-if="status === 'error'" class="state err">{{ errorMessage }}</span>
    </div>

    <div
      class="stage"
      :class="[layoutMode, { solo: layoutMode === 'speaker' && !showSpeakerSide }]"
    >
      <!-- 无人出画面：头像 + 名称墙 -->
      <template v-if="layoutMode === 'avatar'">
        <div class="avatar-wall">
          <div
            v-for="p in avatarParticipants"
            :key="p.identity"
            class="avatar-card"
            :class="{ speaking: p.isSpeaking, local: p.isLocal }"
          >
            <div class="avatar-circle" aria-hidden="true">
              {{ (p.name || '?').slice(0, 1) }}
            </div>
            <div class="avatar-name">
              {{ p.name }}
              <span v-if="p.isLocal">（我）</span>
            </div>
            <div class="avatar-meta">
              <span v-if="p.isHost">主持</span>
              <span v-if="p.isScreenSharing">共享</span>
              <span v-if="!p.isMicrophoneEnabled">静音</span>
            </div>
          </div>
          <div v-if="!avatarParticipants.length" class="empty">
            {{ status === 'connecting' ? '正在进入房间…' : '正在等候参会者' }}
          </div>
        </div>
      </template>

      <!-- 有人开摄像头或投屏：单主视图 + 右侧成员小窗 -->
      <template v-else>
        <div class="speaker-main">
          <div
            v-if="speakerParticipant"
            class="tile main"
            :class="{ speaking: speakerParticipant.isSpeaking && !mainIsScreen }"
          >
            <MediaTrack
              v-if="mainStageTrack"
              :key="`${speakerParticipant.identity}-${mainIsScreen ? 'screen' : 'camera'}`"
              :track="mainStageTrack"
              :mirror="speakerParticipant.isLocal && !mainIsScreen"
              :fit="mainIsScreen ? 'contain' : 'cover'"
              muted
            />
            <div v-else class="placeholder">
              <span class="placeholder-name">{{ speakerParticipant.name }}</span>
            </div>
            <div class="label">
              <span>
                {{ speakerParticipant.name }}
                <span v-if="speakerParticipant.isLocal">（我）</span>
                <span v-if="mainIsScreen"> · 正在共享屏幕</span>
              </span>
            </div>
          </div>
          <div v-else class="empty">暂无画面</div>
        </div>
        <div v-if="showSpeakerSide" class="speaker-side">
          <div
            v-for="p in sideParticipants"
            :key="p.identity"
            class="tile side"
            :class="{ speaking: p.isSpeaking }"
          >
            <MediaTrack
              v-if="videoFor(p).track"
              :key="videoFor(p).isScreen ? 'screen' : 'camera'"
              :track="videoFor(p).track"
              :mirror="p.isLocal && !videoFor(p).isScreen"
              :fit="videoFor(p).isScreen ? 'contain' : 'cover'"
              muted
            />
            <div v-else class="placeholder sm">
              <span class="placeholder-name">{{ p.name }}</span>
            </div>
            <div class="label">
              {{ p.name }}
              <span v-if="videoFor(p).isScreen"> · 共享</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.egress {
  position: fixed;
  inset: 0;
  display: flex;
  flex-direction: column;
  height: 100%;
  color: var(--vc-ink);
  font-family: -apple-system, 'PingFang SC', 'Microsoft YaHei', 'Helvetica Neue', Arial, sans-serif;
  /* 录制固定深色，避免跟随 egress 浏览器主题导致画面不一致 */
  --vc-panel-solid: #111827;
  --vc-line: rgba(148, 163, 184, 0.18);
  --vc-ink: #e8eef7;
  --vc-ink-soft: #d5deeb;
  --vc-muted: #93a4bd;
  --vc-live: #34d399;
  --vc-shadow: 0 12px 32px rgba(0, 0, 0, 0.28);
  --vc-item-hover: rgba(255, 255, 255, 0.06);
  background: #0b1220;
}

.egress-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex: 0 0 auto;
  padding: 14px 20px;
  background: rgba(0, 0, 0, 0.35);
  border-bottom: 1px solid var(--vc-line);
}

.egress-roomslug {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.room-label {
  font-size: 16px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.room-sub {
  font-size: 12px;
  color: var(--vc-muted);
}

.state {
  font-size: 13px;
  color: var(--vc-muted);
  flex: 0 0 auto;
}

.state.ok {
  color: var(--vc-live);
}

.state.err {
  color: #f87171;
}

.stage {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 14px 16px 8px;
  display: flex;
  flex-direction: column;
}

.stage.speaker {
  display: grid;
  grid-template-columns: 1fr 180px;
  gap: 12px;
}

.stage.speaker.solo {
  grid-template-columns: 1fr;
}

.avatar-wall {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-content: center;
  justify-content: center;
  overflow-y: auto;
  padding: 24px 12px;
}

.avatar-card {
  width: 140px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 16px 12px;
  border-radius: 16px;
  background: var(--vc-panel-solid);
  border: 2px solid var(--vc-line);
  box-shadow: var(--vc-shadow);
  color: var(--vc-ink);
  transition: border-color 0.15s ease;
}

.avatar-card.speaking {
  border-color: var(--vc-live);
}

.avatar-card.local {
  background: var(--vc-item-hover);
}

.avatar-circle {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: 28px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(145deg, #f3a04c, #ef4444 75%);
}

.avatar-name {
  font-size: 14px;
  font-weight: 500;
  text-align: center;
  word-break: break-word;
  max-width: 100%;
  color: var(--vc-ink);
}

.avatar-meta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 11px;
  color: var(--vc-muted);
  min-height: 16px;
}

.speaker-main {
  min-height: 0;
  height: 100%;
}

.speaker-side {
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
  min-height: 0;
  height: 100%;
}

.tile {
  position: relative;
  min-height: 0;
  height: 100%;
  background: var(--vc-panel-solid);
  border-radius: 14px;
  overflow: hidden;
  border: 2px solid var(--vc-line);
  box-shadow: var(--vc-shadow);
  transition: border-color 0.15s ease;
}

.tile.main {
  height: 100%;
  min-height: 0;
}

.tile.side {
  aspect-ratio: 16 / 10;
  height: auto;
  flex-shrink: 0;
}

.tile.speaking {
  border-color: var(--vc-live);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--vc-live) 35%, transparent);
}

.tile :deep(video) {
  width: 100%;
  height: 100%;
  border-radius: 0;
}

.placeholder {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
  padding: 16px;
  color: var(--vc-muted);
  background: var(--vc-item-hover);
}

.placeholder-name {
  max-width: 100%;
  font-size: clamp(18px, 3.2vw, 36px);
  font-weight: 600;
  line-height: 1.3;
  text-align: center;
  word-break: break-word;
  color: var(--vc-ink-soft);
}

.placeholder.sm .placeholder-name {
  font-size: 14px;
}

.label {
  position: absolute;
  left: 10px;
  bottom: 10px;
  padding: 4px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--vc-panel-solid) 82%, transparent);
  color: var(--vc-ink);
  border: 1px solid var(--vc-line);
  backdrop-filter: blur(8px);
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  max-width: calc(100% - 20px);
}

.empty {
  color: var(--vc-muted);
  padding: 40px 0;
}
</style>
