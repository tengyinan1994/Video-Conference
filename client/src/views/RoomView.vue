<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Alert,
  Badge,
  Button,
  Drawer,
  Input,
  Modal,
  Select,
  Space,
  Tag,
  Tooltip,
  message,
} from 'ant-design-vue'
import {
  AudioMutedOutlined,
  AudioOutlined,
  ClockCircleOutlined,
  CopyOutlined,
  DesktopOutlined,
  ExpandOutlined,
  AppstoreOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  LinkOutlined,
  LogoutOutlined,
  MessageOutlined,
  SoundOutlined,
  TeamOutlined,
  UserDeleteOutlined,
  VideoCameraAddOutlined,
  VideoCameraOutlined,
  VideoCameraFilled,
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import MediaTrack from '@/components/MediaTrack.vue'
import QualityBars from '@/components/QualityBars.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import {
  kickParticipant,
  muteAllParticipants,
  unmuteAllParticipants,
  claimHost,
  startRecording,
  stopRecording,
  recordingStatus,
  shareView,
  createToken,
  type MeetingShareView,
} from '@/api/conference'
import { ApiError } from '@/utils/request'
import { isLoggedIn } from '@/stores/auth'
import {
  useLiveKitRoom,
  mediaErrorMessage,
  collectRemoteAudioTracks,
  type AttachableTrack,
  type MediaParticipant,
} from '@/composables/useLiveKitRoom'
import {
  readMeetingSession,
  writeMeetingSession,
  removeMeetingSession,
  type MeetingSession as SessionPayload,
} from '@/utils/meetingSession'

const route = useRoute()
const router = useRouter()
const {
  status,
  errorMessage,
  participants,
  micEnabled,
  cameraEnabled,
  screenSharing,
  chatMessages,
  layoutMode,
  speakerParticipant,
  showSpeakerSide,
  activeVideoCount,
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
} = useLiveKitRoom()

const session = ref<SessionPayload | null>(null)
const joining = ref(false)
const memberOpen = ref(false)
const chatOpen = ref(false)
const chatDraft = ref('')
const chatListEl = ref<HTMLElement | null>(null)
/** 聊天面板关闭期间收到的远端消息数（本地发送不计） */
const chatUnread = ref(0)
const hostActing = ref(false)
const muteAllActive = ref(false)
const recordingActive = ref(false)
const recordingActing = ref(false)
const inviteOpen = ref(false)
const inviteLoading = ref(false)
const inviteInfo = ref<MeetingShareView | null>(null)
let recordingPollTimer: ReturnType<typeof setInterval> | null = null
/** 是否已取得一次可靠的录制状态快照（用于边沿检测，避免进房时误报） */
let recordingStatusInited = false
/** 是否已提示过「会议正在录制」（新进成员进房时提示一次，避免轮询重复弹） */
let joinRecordingNotified = false

// —— 会议已进行时长（正向计时）——
// 计时起点 = actualStartAt ?? startAt：首个参会者提前入会时后端记录 actualStartAt，
// 从入会那一刻起累计；无人提前入会则按预定开始时间 startAt 到点起算；
// 两者都没有时退回本端首次进房时刻（localStorage）。大厅「已进行」进度也是同一套语义。
let meetingTimer: ReturnType<typeof setInterval> | null = null
const meetingStartAt = ref<number | null>(null)
const meetingElapsed = ref(0)

const meetingStartKey = computed(() => `vc.meetingStart.${String(route.params.room)}`)

function resolveMeetingStartMs(): number {
  const actualStart = session.value?.actualStartAt
  if (actualStart) {
    const ms = dayjs(actualStart).valueOf()
    if (Number.isFinite(ms) && ms > 0) return ms
  }
  const fromSession = session.value?.startAt
  if (fromSession) {
    const ms = dayjs(fromSession).valueOf()
    if (Number.isFinite(ms) && ms > 0) return ms
  }
  const persisted = Number(localStorage.getItem(meetingStartKey.value))
  if (persisted > 0) return persisted
  const now = Date.now()
  localStorage.setItem(meetingStartKey.value, String(now))
  return now
}

function ensureMeetingStart() {
  meetingStartAt.value = resolveMeetingStartMs()
}

function tickMeetingElapsed() {
  if (meetingStartAt.value == null) return
  meetingElapsed.value = Math.max(0, Math.floor((Date.now() - meetingStartAt.value) / 1000))
}

function startMeetingTimer() {
  ensureMeetingStart()
  tickMeetingElapsed()
  if (meetingTimer) return
  meetingTimer = setInterval(tickMeetingElapsed, 1000)
}

function stopMeetingTimer() {
  if (meetingTimer) {
    clearInterval(meetingTimer)
    meetingTimer = null
  }
}

const elapsedText = computed(() => {
  const s = meetingElapsed.value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(Math.floor(s / 3600))}:${pad(Math.floor((s % 3600) / 60))}:${pad(s % 60)}`
})

// —— 录制时长（正向计时）——
// 以当前进行中录制段的 startedAt（服务端 LiveKit egress 启动时间）为起点，
// 刷新页面/短暂重连/其他参会者都会看到同一段录制的真实时长，而非本端局部计时归零。
const recordingStartedAt = ref<number | null>(null)
const recordingElapsed = ref(0)
let recordingTimer: ReturnType<typeof setInterval> | null = null

function tickRecordingElapsed() {
  if (recordingStartedAt.value === null) return
  recordingElapsed.value = Math.max(0, Math.floor((Date.now() - recordingStartedAt.value) / 1000))
}

function startRecordingTimer() {
  tickRecordingElapsed()
  if (recordingTimer) return
  recordingTimer = setInterval(tickRecordingElapsed, 1000)
}

function stopRecordingTimer() {
  if (recordingTimer) {
    clearInterval(recordingTimer)
    recordingTimer = null
  }
}

function syncRecordingTimer() {
  if (recordingActive.value && recordingStartedAt.value !== null) {
    startRecordingTimer()
  } else {
    stopRecordingTimer()
    recordingElapsed.value = 0
  }
}

const recordingElapsedText = computed(() => {
  const s = recordingElapsed.value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(Math.floor(s / 3600))}:${pad(Math.floor((s % 3600) / 60))}:${pad(s % 60)}`
})

/** 取当前进行中的录制段（starting/active 中 seq 最大者，其 startedAt 即本段录制起点） */
function activeRecordingSegment(
  segments: { status: string; seq: number; startedAt?: string }[],
) {
  return [...segments]
    .filter((s) => s.status === 'starting' || s.status === 'active')
    .sort((a, b) => b.seq - a.seq)[0]
}

type InviteKind = 'guest' | 'member'

const statusText = computed(() => {
  switch (status.value) {
    case 'connecting':
      return '连接中…'
    case 'connected':
      return '已连接'
    case 'reconnecting':
      return '重连中…'
    case 'disconnected':
      return '已断开'
    case 'kicked':
      return '已被移出'
    case 'ended':
      return '会议已结束'
    case 'error':
      return '连接失败'
    default:
      return '未连接'
  }
})

const statusTagColor = computed(() => {
  if (status.value === 'connected') return 'success'
  if (status.value === 'kicked' || status.value === 'ended' || status.value === 'error') return 'error'
  if (status.value === 'reconnecting') return 'warning'
  return 'processing'
})

const isHost = computed(() => !!session.value?.isHost)
const canInvite = computed(() => !!session.value?.shareCode)
/** 终态：会议已结束或被移出，不可再重新连接（重复进房只会重建已删除的房间） */
const isTerminal = computed(() => status.value === 'ended' || status.value === 'kicked')

const inviteTime = computed(() => {
  const m = inviteInfo.value
  if (!m?.startAt) return null
  const start = dayjs(m.startAt)
  const end = m.endAt ? dayjs(m.endAt) : null
  const weekdays = ['日', '一', '二', '三', '四', '五', '六']
  const date = `${start.format('YYYY年M月D日')} 周${weekdays[start.day()]}`
  const range = end
    ? end.isSame(start, 'day')
      ? `${start.format('HH:mm')} – ${end.format('HH:mm')}`
      : `${start.format('HH:mm')} – ${end.format('YYYY年M月D日')} ${end.format('HH:mm')}`
    : start.format('HH:mm')
  let duration = ''
  if (end) {
    const mins = end.diff(start, 'minute')
    if (mins > 0) {
      const h = Math.floor(mins / 60)
      const mRemain = mins % 60
      duration = h > 0 ? (mRemain ? `${h} 小时 ${mRemain} 分钟` : `${h} 小时`) : `${mins} 分钟`
    }
  }
  return { date, range, duration }
})

const inviteEnded = computed(() => {
  const s = inviteInfo.value?.status
  return s === 'ended' || s === 'released'
})
const inviteTimeText = computed(() => {
  const t = inviteTime.value
  if (!t) return '-'
  if (!t.duration) return `${t.date} ${t.range}`
  const label = inviteEnded.value ? '实际时长' : '预计时长'
  return `${t.date} ${t.range}，${label} ${t.duration}`
})

async function syncHostRole() {
  // 主持权仅来自预定人（进房 Token 的 isHost），不再自动接任/抢占
  if (!session.value || status.value !== 'connected') return
  if (!session.value.isHost) return
  try {
    // 预定主持人进房后刷新 LiveKit metadata，便于其他人看到「主持」标记
    await claimHost(session.value.room, session.value.identity)
  } catch {
    // ignore
  }
}

const avatarParticipants = computed(() => participants.value)

/** 远端音频与布局解耦：投屏/单主视图/侧栏隐藏时仍要播放别人说话和共享的标签页声音 */
const remoteAudioTracks = computed(() => collectRemoteAudioTracks(participants.value))

/** 侧栏只列出有画面的成员，便于切换主视图 */
const sideParticipants = computed(() =>
  participants.value.filter((p) => p.isCameraEnabled || p.isScreenSharing),
)

/** 多人有画面时，可手动收起侧栏为单主视图 */
const preferSoloMain = ref(false)
const canToggleSoloLayout = computed(
  () => layoutMode.value === 'speaker' && activeVideoCount.value > 1,
)
const sideVisible = computed(() => showSpeakerSide.value && !preferSoloMain.value)

watch(activeVideoCount, (n, prev) => {
  if (n <= 1) {
    preferSoloMain.value = false
    return
  }
  // 有新人出画面（1→多人）：自动展开侧栏宫格，便于切换成员
  if (typeof prev === 'number' && n > prev) {
    preferSoloMain.value = false
  }
})

function toggleSoloLayout() {
  if (!canToggleSoloLayout.value) return
  preferSoloMain.value = !preferSoloMain.value
}

const mainStageEl = ref<HTMLElement | null>(null)
const mainFullscreen = ref(false)

function syncMainFullscreen() {
  const el = mainStageEl.value
  mainFullscreen.value = !!el && document.fullscreenElement === el
}

async function toggleMainFullscreen() {
  const el = mainStageEl.value
  if (!el) return
  try {
    if (document.fullscreenElement === el) {
      await document.exitFullscreen()
    } else {
      if (document.fullscreenElement) {
        await document.exitFullscreen()
      }
      await el.requestFullscreen()
    }
  } catch {
    message.warning('当前环境无法进入全屏')
  } finally {
    syncMainFullscreen()
  }
}

/** 该轨当前是否可渲染：必须有 mediaStreamTrack 且状态为 live。
 *  无 MST 或已 ended 的轨道挂到 <video> 上只会渲染成一帧黑屏，必须跳过。 */
function canRenderTrack(track?: AttachableTrack | null): boolean {
  if (!track) return false
  const mst = track.mediaStreamTrack
  return !!mst && mst.readyState === 'live'
}

/** 取参与人当前可渲染的最佳视频轨：投屏轨若已失效则回退到摄像头轨，避免黑屏。 */
function pickVideoTrack(p?: MediaParticipant | null): { track?: AttachableTrack; isScreen: boolean } {
  if (!p) return { track: undefined, isScreen: false }
  if (canRenderTrack(p.screenTrack)) return { track: p.screenTrack, isScreen: true }
  if (canRenderTrack(p.cameraTrack)) return { track: p.cameraTrack, isScreen: false }
  return { track: undefined, isScreen: false }
}

const mainVideo = computed(() => {
  const p = speakerParticipant.value
  if (!p) return null
  const best = pickVideoTrack(p)
  return { participant: p, track: best.track, isScreen: best.isScreen }
})
const mainStageTrack = computed(() => mainVideo.value?.track)
const mainIsScreen = computed(() => !!mainVideo.value?.isScreen)

/** 侧栏 tile 与主画面共用同一套"取可渲染 live 轨"逻辑，避免投屏轨失效后渲染黑屏 */
function videoFor(p: MediaParticipant) {
  return pickVideoTrack(p)
}

const micOptions = computed(() =>
  audioInputs.value.map((d) => ({ value: d.deviceId, label: d.label })),
)
const cameraOptions = computed(() =>
  videoInputs.value.map((d) => ({ value: d.deviceId, label: d.label })),
)
const speakerOptions = computed(() =>
  audioOutputs.value.map((d) => ({ value: d.deviceId, label: d.label })),
)

async function refreshSessionToken(parsed: SessionPayload): Promise<SessionPayload | null> {
  const nick = (parsed.nickname || '').trim()
  if (!nick) return null
  try {
    const data = parsed.shareCode
      ? await createToken({ shareCode: parsed.shareCode, nickname: nick })
      : await createToken({ room: parsed.room, nickname: nick })
    if (data.room !== parsed.room && data.room !== route.params.room) {
      return null
    }
    const next: SessionPayload = {
      ...parsed,
      serverUrl: data.serverUrl,
      token: data.token,
      room: data.room,
      title: data.title || parsed.title,
      identity: data.identity,
      nickname: data.nickname || nick,
      expiresAt: data.expiresAt,
      isHost: !!data.isHost,
      recordEnabled: !!data.recordEnabled,
      recordingActive: !!data.recordingActive,
      startAt: data.startAt || parsed.startAt,
      actualStartAt: data.actualStartAt || parsed.actualStartAt,
    }
    writeMeetingSession(next)
    return next
  } catch (err) {
    const msg = err instanceof ApiError ? err.message : '重新获取进房凭证失败'
    message.error(msg)
    return null
  }
}

async function enter() {
  const room = String(route.params.room)
  let parsed = readMeetingSession(room)
  if (!parsed) {
    message.warning('缺少进房凭证，请重新加入')
    await leaveToEntry()
    return
  }
  if (parsed.room !== room) {
    message.warning('房间不匹配，请重新加入')
    await leaveToEntry()
    return
  }
  // 进房前始终向后端重新校验会议是否仍可加入并换取新凭证（createToken 会走 assertJoinable）。
  // 不能只信任本地未过期的缓存 Token：主持人结束会议后 LiveKit 房间已被删除，
  // 复用旧 Token 会被 LiveKit 按 Token 重建房间，导致「已结束的会议仍能连接并继续计时」。
  const renewed = await refreshSessionToken(parsed)
  if (!renewed) {
    removeMeetingSession(room)
    await leaveToEntry(parsed)
    return
  }
  parsed = renewed
  session.value = parsed
  recordingActive.value = !!parsed.recordingActive
  chatUnread.value = 0
  joining.value = true
  try {
    await connect(parsed.serverUrl, parsed.token, {
      enableMic: !!parsed.enableMic,
      enableCamera: !!parsed.enableCamera,
    })
    await refreshDevices()
    await syncHostRole()
    await refreshRecordingStatus()
    startRecordingPoll()
  } catch {
    // 信令失败时换新 Token 再试一次（主持人刚离开 / 票据失效等）
    const renewed = await refreshSessionToken(parsed)
    if (renewed) {
      session.value = renewed
      recordingActive.value = !!renewed.recordingActive
      try {
        await connect(renewed.serverUrl, renewed.token, {
          enableMic: !!renewed.enableMic,
          enableCamera: !!renewed.enableCamera,
        })
        await refreshDevices()
        await syncHostRole()
        await refreshRecordingStatus()
        startRecordingPoll()
        return
      } catch {
        // errorMessage already set by second attempt
      }
    }
  } finally {
    joining.value = false
  }
}

async function retryEnter() {
  if (joining.value || status.value === 'connecting') return
  if (status.value === 'ended') {
    message.warning('会议已结束，无法重新连接')
    return
  }
  if (status.value === 'kicked') {
    message.warning('你已被主持人移出会议，无法重新连接')
    return
  }
  await enter()
}

async function leaveToEntry(payload?: SessionPayload | null) {
  const s = payload ?? session.value
  if (s?.fromShare && s.shareCode) {
    await router.replace({ name: 'join', params: { shareCode: s.shareCode } })
    return
  }
  if (isLoggedIn()) {
    await router.replace({ name: 'lobby' })
  } else {
    await router.replace({ name: 'login' })
  }
}

async function leave() {
  stopRecordingPoll()
  stopRecordingTimer()
  await disconnect()
  const s = session.value
  removeMeetingSession(s?.room || route.params.room)
  await leaveToEntry(s)
}

function shareLink() {
  const code = session.value?.shareCode
  if (!code) return ''
  return `${window.location.origin}/join/${code}`
}

async function openInvite() {
  if (!session.value?.shareCode) {
    message.warning('当前会议暂无邀请链接')
    return
  }
  inviteOpen.value = true
  inviteLoading.value = true
  inviteInfo.value = null
  try {
    inviteInfo.value = await shareView(session.value.shareCode)
  } catch {
    // 无详情时仍可复制链接
  } finally {
    inviteLoading.value = false
  }
}

function clearInvite() {
  inviteInfo.value = null
}

function buildInviteText(kind: InviteKind) {
  const title = inviteInfo.value?.title || session.value?.title || '视频会议'
  const hostName = inviteInfo.value?.hostName || '-'
  const how =
    kind === 'guest'
      ? '打开下方链接，填写昵称即可进入（无需账号）'
      : '请使用公司账号登录后，通过下方链接进入会议'
  return [
    '【视频会议邀请】',
    `主题：${title}`,
    `主持人：${hostName}`,
    `时间：${inviteTimeText.value}`,
    `加入方式：${how}`,
    `会议链接：${shareLink()}`,
  ].join('\n')
}

async function copyInvite(kind: InviteKind) {
  if (!session.value?.shareCode) return
  const text = buildInviteText(kind)
  try {
    await navigator.clipboard.writeText(text)
    message.success(kind === 'guest' ? '游客邀请已复制' : '同事邀请已复制')
  } catch {
    message.info(text)
  }
}

async function onSendChat() {
  const text = chatDraft.value
  if (!text.trim()) return
  try {
    await sendChat(text)
    chatDraft.value = ''
  } catch (err) {
    message.error(err instanceof Error ? err.message : '发送失败')
  }
}

async function onKick(identity: string, name: string) {
  if (!session.value) return
  hostActing.value = true
  try {
    await kickParticipant(session.value.room, identity, session.value.identity)
    message.success(`已踢出 ${name}`)
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '踢人失败')
  } finally {
    hostActing.value = false
  }
}

async function onMuteAll() {
  if (!session.value) return
  hostActing.value = true
  try {
    const res = await muteAllParticipants(session.value.room, session.value.identity)
    message.success(`已全员静音（${res.mutedCount} 路麦克风）`)
    muteAllActive.value = true
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '全员静音失败')
  } finally {
    hostActing.value = false
  }
}

async function onUnmuteAll() {
  if (!session.value) return
  hostActing.value = true
  try {
    const res = await unmuteAllParticipants(session.value.room, session.value.identity)
    message.success(`已取消全员静音（${res.unmutedCount} 路麦克风）`)
    muteAllActive.value = false
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '取消全员静音失败')
  } finally {
    hostActing.value = false
  }
}

async function refreshRecordingStatus() {
  if (!session.value || recordingActing.value) return
  try {
    const res = await recordingStatus({ room: session.value.room })
    if (recordingActing.value) return
    const active = !!res.active
    // 新进会议成员：拿到首个可靠状态后，若会议正在录制则提示一次「会议正在录制」；
    // 仅在进房首轮（recordingStatusInited=false）判定，避免轮询重复弹，也避免与会中「开始/停止」边沿混用。
    if (!recordingStatusInited && active && !joinRecordingNotified) {
      message.warning('会议正在录制')
      joinRecordingNotified = true
    }
    // 录制状态跨端边沿检测：进房首轮（recordingStatusInited=false）只处理上面的「进房即在录制」提示，不判边沿；
    // 本机主持人由 onToggleRecording 点击即时提示，轮询跳过避免双报；其余参会者补上「开始/停止录制」提示。
    if (recordingStatusInited && active !== recordingActive.value && !isHost.value) {
      message.success(active ? '已开始录制' : '已停止录制')
    }
    recordingActive.value = active
    const seg = activeRecordingSegment(res.segments || [])
    recordingStartedAt.value = active && seg?.startedAt ? dayjs(seg.startedAt).valueOf() : null
    syncRecordingTimer()
    recordingStatusInited = true
  } catch {
    // ignore poll errors
  }
}

function startRecordingPoll() {
  stopRecordingPoll()
  recordingPollTimer = setInterval(() => {
    void refreshRecordingStatus()
  }, 2000)
}

function stopRecordingPoll() {
  if (recordingPollTimer) {
    clearInterval(recordingPollTimer)
    recordingPollTimer = null
  }
}

async function onToggleRecording() {
  if (!session.value || !isHost.value || recordingActing.value) return
  const room = session.value.room
  const stopping = recordingActive.value
  recordingActive.value = !stopping
  recordingActing.value = true
  try {
    if (stopping) {
      await stopRecording(room)
      message.success('已停止录制')
    } else {
      const seg = await startRecording(room)
      message.success('已开始录制')
      recordingStartedAt.value = seg?.startedAt ? dayjs(seg.startedAt).valueOf() : null
    }
  } catch (err) {
    recordingActive.value = stopping
    message.error(err instanceof ApiError ? err.message : '录制操作失败')
    await refreshRecordingStatus()
  } finally {
    recordingActing.value = false
    syncRecordingTimer()
  }
}

async function onToggleMic() {
  try {
    await toggleMic()
  } catch (err) {
    message.error(mediaErrorMessage(err))
  }
}

async function onToggleCamera() {
  try {
    await toggleCamera()
  } catch (err) {
    message.error(mediaErrorMessage(err))
  }
}

async function onToggleScreenShare() {
  try {
    await toggleScreenShare()
  } catch (err) {
    message.error(mediaErrorMessage(err))
  }
}

watch(
  chatMessages,
  async () => {
    await nextTick()
    if (chatListEl.value) {
      chatListEl.value.scrollTop = chatListEl.value.scrollHeight
    }
  },
  { deep: true },
)

// 聊天面板关闭时收到远端消息才累计未读；本地发送的不计
watch(
  chatMessages,
  (list, prev) => {
    if (chatOpen.value) return
    const prevIds = new Set((prev ?? []).map((m) => m.id))
    const addedRemote = list.filter((m) => !prevIds.has(m.id) && !m.isLocal)
    if (addedRemote.length) chatUnread.value += addedRemote.length
  },
  { deep: true },
)

// 打开聊天面板即视为已读
watch(chatOpen, (open) => {
  if (open) chatUnread.value = 0
})

watch(status, (s) => {
  if (s === 'connected') {
    startMeetingTimer()
    return
  }
  if (s === 'kicked') {
    stopMeetingTimer()
    message.warning('你已被主持人移出会议')
  } else if (s === 'ended') {
    stopMeetingTimer()
    message.warning('会议已结束')
  }
})

// 会议房里实时把麦克风/摄像头开关写回 session，刷新页面 / 短暂重连后仍保留当前状态，
// 避免 `enter()` 只读到进会那一刻的 enableMic/enableCamera 而回退开关。
watch(
  [micEnabled, cameraEnabled],
  ([mic, cam]) => {
    const s = session.value
    if (!s) return
    // 仅连接态才写回会话：断连/卸载时本地摄像头/麦克风轨道会被解发布而瞬间变为「关」，
    // 此时写回会把刷新前的正确状态覆盖成「关闭」（进房前开了摄像头，刷新后却变关）。
    if (status.value !== 'connected') return
    if (s.enableMic === mic && s.enableCamera === cam) return
    s.enableMic = mic
    s.enableCamera = cam
    writeMeetingSession(s)
  },
)

onMounted(() => {
  document.addEventListener('fullscreenchange', syncMainFullscreen)
  void enter()
})

onBeforeUnmount(() => {
  stopRecordingPoll()
  stopMeetingTimer()
  stopRecordingTimer()
  document.removeEventListener('fullscreenchange', syncMainFullscreen)
  if (document.fullscreenElement === mainStageEl.value) {
    void document.exitFullscreen().catch(() => undefined)
  }
})
</script>

<template>
  <div class="room">
    <header class="top">
      <div class="top-left">
        <strong>{{ session?.title || `房间 ${route.params.room}` }}</strong>
        <Tag class="tag" :color="statusTagColor">
          {{ statusText }}
        </Tag>
        <Tag v-if="isHost" color="gold">主持人</Tag>
        <Tag v-if="recordingActive" color="red" class="tag recording">
          录制中<span v-if="recordingStartedAt != null" class="recording-time">{{ recordingElapsedText }}</span>
        </Tag>
        <Tag v-if="status === 'connected'" class="tag duration" color="blue">
          <ClockCircleOutlined class="duration-icon" />
          已进行 {{ elapsedText }}
        </Tag>
      </div>
      <div class="top-right">
        <div v-if="session" class="meta">
          {{ session.nickname }}
        </div>
        <ThemeToggle />
      </div>
    </header>

    <Alert
      v-if="errorMessage"
      type="error"
      show-icon
      :message="errorMessage"
      class="banner"
    >
      <template #action>
        <Button v-if="!isTerminal" size="small" type="primary" :loading="joining" @click="retryEnter">
          重新连接
        </Button>
        <Button v-else size="small" type="primary" @click="leave">离开会议</Button>
      </template>
    </Alert>
    <Alert
      v-else-if="status === 'reconnecting'"
      type="warning"
      show-icon
      message="网络波动，正在重连…"
      class="banner"
    />

    <!-- 远端音频始终挂载，避免投屏/单主视图把侧栏卸掉后听不见人 -->
    <div class="remote-audio" aria-hidden="true">
      <MediaTrack
        v-for="item in remoteAudioTracks"
        :key="item.key"
        :track="item.track"
      />
    </div>

    <div class="stage" :class="[layoutMode, { solo: layoutMode === 'speaker' && !sideVisible }]">
      <!-- 无人出画面：头像 + 名称墙 -->
      <template v-if="layoutMode === 'avatar'">
        <div class="avatar-wall">
          <div
            v-for="p in avatarParticipants"
            :key="p.identity"
            class="avatar-card"
            :class="{ speaking: p.isSpeaking, local: p.isLocal }"
          >
            <div class="avatar-circle" aria-hidden="true">{{ p.name.slice(0, 1) }}</div>
            <div class="avatar-name">
              {{ p.name }}
              <span v-if="p.isLocal">（我）</span>
            </div>
            <div class="avatar-meta">
              <span v-if="p.isHost">主持</span>
              <span v-if="!p.isMicrophoneEnabled">静音</span>
              <QualityBars :quality="p.connectionQuality" />
            </div>
          </div>
          <div v-if="!avatarParticipants.length" class="empty">
            {{ joining || status === 'connecting' ? '正在进入房间…' : '暂无参与者' }}
          </div>
        </div>
      </template>

      <!-- 有人开摄像头或投屏：单主视图；多人有画面时右侧可切换 -->
      <template v-else>
        <div class="speaker-main">
          <div
            v-if="speakerParticipant"
            ref="mainStageEl"
            class="tile main"
            :class="{
              speaking: speakerParticipant.isSpeaking && !mainIsScreen,
              fullscreen: mainFullscreen,
            }"
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
            <div class="main-actions">
              <Tooltip
                v-if="canToggleSoloLayout"
                :title="preferSoloMain ? '恢复成员侧栏' : '切换为单主视图'"
              >
                <button
                  type="button"
                  class="layout-toggle"
                  :aria-label="preferSoloMain ? '恢复成员侧栏' : '切换为单主视图'"
                  @click.stop="toggleSoloLayout"
                >
                  <ExpandOutlined v-if="!preferSoloMain" />
                  <AppstoreOutlined v-else />
                </button>
              </Tooltip>
              <Tooltip :title="mainFullscreen ? '退出全屏' : '主视图全屏'">
                <button
                  type="button"
                  class="layout-toggle"
                  :aria-label="mainFullscreen ? '退出全屏' : '主视图全屏'"
                  @click.stop="toggleMainFullscreen"
                >
                  <FullscreenExitOutlined v-if="mainFullscreen" />
                  <FullscreenOutlined v-else />
                </button>
              </Tooltip>
            </div>
            <div class="label">
              <span>
                {{ speakerParticipant.name }}
                <span v-if="speakerParticipant.isLocal">（我）</span>
                <span v-if="mainIsScreen"> · 正在共享屏幕</span>
              </span>
              <QualityBars :quality="speakerParticipant.connectionQuality" />
            </div>
          </div>
          <div v-else class="empty">暂无画面</div>
        </div>
        <div v-if="sideVisible" class="speaker-side">
          <p v-if="sideParticipants.length" class="side-hint">点击成员切换主视图</p>
          <div
            v-for="p in sideParticipants"
            :key="p.identity"
            class="tile side clickable"
            :class="{
              speaking: p.isSpeaking,
              active: speakerParticipant?.identity === p.identity,
            }"
            :title="`将 ${p.name} 设为主视图`"
            @click="focusParticipant(p.identity)"
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

    <footer class="controls">
      <div class="devices">
        <label class="device-field">
          <span class="device-label">麦克风</span>
          <Select
            :value="selectedMicId"
            :options="micOptions"
            placeholder="选择麦克风"
            style="width: 180px"
            size="small"
            @change="(v: any) => switchMic(String(v))"
          />
        </label>
        <label class="device-field">
          <span class="device-label">摄像头</span>
          <Select
            :value="selectedCameraId"
            :options="cameraOptions"
            placeholder="选择摄像头"
            style="width: 180px"
            size="small"
            @change="(v: any) => switchCamera(String(v))"
          />
        </label>
        <label class="device-field">
          <span class="device-label">扬声器</span>
          <Tooltip v-if="!speakerSupported" title="当前浏览器不支持切换扬声器（setSinkId）">
            <Select
              :value="selectedSpeakerId"
              :options="speakerOptions"
              placeholder="选择扬声器"
              style="width: 180px"
              size="small"
              disabled
            />
          </Tooltip>
          <Select
            v-else
            :value="selectedSpeakerId"
            :options="speakerOptions"
            placeholder="选择扬声器"
            style="width: 180px"
            size="small"
            @change="(v: any) => switchSpeaker(String(v))"
          />
        </label>
      </div>

      <Space wrap>
        <Button @click="onToggleMic">
          <template #icon>
            <AudioMutedOutlined v-if="!micEnabled" />
            <AudioOutlined v-else />
          </template>
          {{ micEnabled ? '静音' : '取消静音' }}
        </Button>
        <Button @click="onToggleCamera">
          <template #icon>
            <VideoCameraOutlined v-if="cameraEnabled" />
            <VideoCameraAddOutlined v-else />
          </template>
          {{ cameraEnabled ? '关摄像头' : '开摄像头' }}
        </Button>
        <Button
          :type="screenSharing ? 'primary' : 'default'"
          :danger="screenSharing"
          @click="onToggleScreenShare"
        >
          <template #icon>
            <DesktopOutlined />
          </template>
          {{ screenSharing ? '停止共享' : '屏幕共享' }}
        </Button>
        <Button @click="memberOpen = true">
          <template #icon>
            <TeamOutlined />
          </template>
          成员 ({{ participants.length }})
        </Button>
        <Button class="chat-btn" @click="chatOpen = true">
          <template #icon>
            <Badge :count="chatUnread" :offset="[5, -2]" size="small">
              <MessageOutlined />
            </Badge>
          </template>
          聊天
        </Button>
        <Button v-if="canInvite" @click="openInvite">
          <template #icon>
            <LinkOutlined />
          </template>
          邀请
        </Button>
        <Button
          v-if="isHost"
          :loading="hostActing"
          :danger="muteAllActive"
          @click="muteAllActive ? onUnmuteAll() : onMuteAll()"
        >
          <template #icon>
            <SoundOutlined v-if="!muteAllActive" />
            <AudioMutedOutlined v-else />
          </template>
          {{ muteAllActive ? '取消全员静音' : '全员静音' }}
        </Button>
        <Button
          v-if="isHost"
          :type="recordingActive ? 'primary' : 'default'"
          :danger="recordingActive"
          :loading="recordingActing"
          @click="onToggleRecording"
        >
          <template #icon>
            <VideoCameraFilled />
          </template>
          {{ recordingActive ? '停止录制' : '开始录制' }}
        </Button>
        <Button danger type="primary" @click="leave">
          <template #icon>
            <LogoutOutlined />
          </template>
          离开
        </Button>
      </Space>
    </footer>

    <Drawer
      v-model:open="memberOpen"
      title="成员列表"
      placement="right"
      :width="320"
    >
      <div v-for="p in participants" :key="p.identity" class="member-row">
        <div class="member-info">
          <div class="member-name">
            {{ p.name }}
            <Tag v-if="p.isLocal" color="blue">我</Tag>
            <Tag v-if="p.isHost" color="gold">主持</Tag>
            <QualityBars :quality="p.connectionQuality" />
          </div>
          <div class="member-sub">
            {{ p.isMicrophoneEnabled ? '麦克风开' : '已静音' }}
            ·
            {{ p.isCameraEnabled ? '摄像头开' : '摄像头关' }}
          </div>
        </div>
        <Button
          v-if="isHost && !p.isLocal"
          size="small"
          danger
          :loading="hostActing"
          @click="onKick(p.identity, p.name)"
        >
          <template #icon>
            <UserDeleteOutlined />
          </template>
          踢出
        </Button>
      </div>
    </Drawer>

    <Drawer
      v-model:open="chatOpen"
      title="聊天"
      placement="right"
      :width="360"
      :body-style="{ padding: 0, display: 'flex', flexDirection: 'column', height: 'calc(100% - 55px)' }"
    >
      <div class="chat-panel">
        <div ref="chatListEl" class="chat-list">
          <div
            v-for="m in chatMessages"
            :key="m.id"
            class="chat-row"
            :class="{ mine: m.isLocal }"
          >
            <div class="chat-meta">
              <span class="chat-name">{{ m.name }}</span>
              <span class="chat-time">{{ new Date(m.ts).toLocaleTimeString() }}</span>
            </div>
            <div class="chat-bubble">{{ m.text }}</div>
          </div>
          <div v-if="!chatMessages.length" class="chat-empty">还没有消息，打个招呼吧</div>
        </div>
        <div class="chat-input">
          <Input
            v-model:value="chatDraft"
            placeholder="输入消息，Enter 发送"
            @pressEnter="onSendChat"
          />
          <Button type="primary" @click="onSendChat">发送</Button>
        </div>
      </div>
    </Drawer>

    <Modal
      v-model:open="inviteOpen"
      :title="inviteInfo?.title || session?.title || '邀请'"
      :footer="null"
      wrap-class-name="room-invite-modal"
      destroy-on-close
      :afterClose="clearInvite"
    >
      <div class="invite-panel">
        <div v-if="inviteLoading" class="invite-loading">加载会议信息…</div>
        <template v-else>
          <div class="invite-fields">
            <div class="invite-line">
              <span class="invite-label">主持人</span>
              <span class="invite-value">{{ inviteInfo?.hostName || '-' }}</span>
            </div>
            <div class="invite-line">
              <span class="invite-label">时间</span>
              <div v-if="inviteTime" class="invite-time">
                <span class="invite-time-date">{{ inviteTime.date }}</span>
                <span class="invite-time-range">
                  <ClockCircleOutlined class="invite-time-icon" />
                  <span class="tabular">{{ inviteTime.range }}</span>
                  <span v-if="inviteTime.duration" class="invite-time-dur">
                    {{ inviteEnded ? '实际时长' : '预计时长' }} {{ inviteTime.duration }}
                  </span>
                </span>
              </div>
              <span v-else class="invite-value">-</span>
            </div>
          </div>

          <div class="invite-actions">
            <div class="invite-action">
              <Button type="primary" class="invite-btn invite-btn-guest" block @click="copyInvite('guest')">
                <template #icon><CopyOutlined /></template>
                复制游客邀请
              </Button>
              <p class="invite-hint">对方打开链接填写昵称即可进入，无需账号</p>
            </div>
            <div class="invite-action">
              <Button class="invite-btn invite-btn-member" block @click="copyInvite('member')">
                <template #icon><CopyOutlined /></template>
                复制同事邀请
              </Button>
              <p class="invite-hint">对方需用账号登录后，通过链接进入会议</p>
            </div>
          </div>
        </template>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.room {
  --brand: #f3a04c;
  height: 100dvh;
  max-height: 100dvh;
  overflow: hidden;
  color: var(--vc-ink);
  display: flex;
  flex-direction: column;
  background: var(--vc-page-grad);
  font-family: -apple-system, 'PingFang SC', 'Microsoft YaHei', Inter, sans-serif;
}
.top {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin: 12px 16px 0;
  padding: 12px 16px;
  border-radius: 14px;
  border: 1px solid var(--vc-line);
  background: var(--vc-panel);
  backdrop-filter: blur(12px);
  box-shadow: var(--vc-shadow);
}
.top-left {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  color: var(--vc-ink);
}
.top-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.tag {
  margin-left: 8px;
}
.tag.duration {
  font-variant-numeric: tabular-nums;
}
.tag.recording {
  font-variant-numeric: tabular-nums;
}
.recording-time {
  margin-left: 4px;
}
.duration-icon {
  margin-right: 4px;
  font-size: 12px;
}
.meta {
  color: var(--vc-muted);
  font-size: 13px;
}
.banner {
  flex-shrink: 0;
  margin: 12px 16px 0;
}
.remote-audio {
  /* 视觉隐藏即可；勿用 display:none / 0x0，部分浏览器会停播 audio */
  position: fixed;
  left: 0;
  top: 0;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  border: 0;
  pointer-events: none;
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
.side-hint {
  margin: 0;
  font-size: 11px;
  color: var(--vc-muted);
  text-align: center;
}
.tile.side.clickable {
  cursor: pointer;
}
.tile.side.clickable:hover {
  border-color: var(--brand);
}
.tile.side.active {
  border-color: var(--brand);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--brand) 35%, transparent);
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
.tile.main.fullscreen {
  border-radius: 0;
  border: none;
  background: #000;
}
.main-actions {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 3;
  display: flex;
  gap: 8px;
}
.layout-toggle {
  width: 34px;
  height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  color: #fff;
  background: rgba(15, 23, 42, 0.55);
  backdrop-filter: blur(6px);
  transition: background 0.15s ease, transform 0.15s ease;
}
.layout-toggle:hover {
  background: rgba(15, 23, 42, 0.78);
  transform: scale(1.04);
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
.controls {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin: 0 16px 16px;
  padding: 12px 16px;
  border-radius: 14px;
  background: var(--vc-panel);
  border: 1px solid var(--vc-line);
  box-shadow: var(--vc-shadow);
  backdrop-filter: blur(12px);
}
.devices {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
  justify-content: center;
}
/* 聊天按钮的图标外包了一层 Badge，antd 的 `.ant-btn > .anticon + span` 间距规则失效，这里补上 */
.chat-btn:deep(.anticon) {
  margin-inline-end: 8px;
}
.device-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.device-label {
  font-size: 12px;
  color: var(--vc-muted);
  line-height: 1;
}
.member-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 10px 0;
  border-bottom: 1px solid var(--vc-line);
  color: var(--vc-ink);
}
.member-name {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  font-weight: 500;
}
.member-sub {
  color: var(--vc-muted);
  font-size: 12px;
  margin-top: 2px;
}
.chat-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  height: 100%;
  background: var(--vc-panel-solid);
  color: var(--vc-ink);
}
.chat-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--vc-bg0);
}
.chat-row {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  max-width: 85%;
}
.chat-row.mine {
  align-self: flex-end;
  align-items: flex-end;
}
.chat-meta {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-bottom: 4px;
  padding: 0 2px;
}
.chat-row.mine .chat-meta {
  flex-direction: row-reverse;
}
.chat-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--vc-muted);
}
.chat-time {
  font-size: 11px;
  color: var(--vc-muted);
  opacity: 0.8;
}
.chat-bubble {
  padding: 8px 12px;
  border-radius: 12px 12px 12px 4px;
  background: var(--vc-panel-solid);
  border: 1px solid var(--vc-line);
  color: var(--vc-ink);
  word-break: break-word;
  line-height: 1.45;
  font-size: 14px;
}
.chat-row.mine .chat-bubble {
  border-radius: 12px 12px 4px 12px;
  /* 文字颜色随主题切换，保证可读性：
     浅色 = 浅橙底 + 深棕字（约 6.7:1）；深色 = 深琥珀底 + 白字（约 5:1） */
  background: var(--vc-brand-fill);
  border-color: var(--vc-brand-fill);
  color: var(--vc-brand-ink);
}
.chat-empty {
  color: var(--vc-muted);
  text-align: center;
  padding: 48px 0;
  margin: auto;
}
.chat-input {
  flex-shrink: 0;
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--vc-line);
  background: var(--vc-panel-solid);
}
.invite-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.invite-loading {
  color: var(--vc-muted);
  font-size: 14px;
  padding: 12px 0 4px;
}
.invite-fields {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.invite-line {
  display: grid;
  grid-template-columns: 52px 1fr;
  gap: 10px;
  align-items: start;
}
.invite-label {
  color: var(--vc-muted);
  font-size: 13px;
  line-height: 1.5;
}
.invite-value {
  color: var(--vc-ink);
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
}
.invite-time {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.invite-time-date {
  color: var(--vc-ink);
  font-size: 14px;
  font-weight: 560;
  line-height: 1.45;
}
.invite-time-range {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  color: var(--vc-muted);
  font-size: 13px;
  line-height: 1.45;
}
.invite-time-icon {
  font-size: 13px;
  color: var(--brand);
}
.invite-time-dur {
  color: var(--vc-muted);
  opacity: 0.85;
}
.invite-time-dur::before {
  content: '';
  display: inline-block;
  width: 1px;
  height: 11px;
  margin-right: 10px;
  vertical-align: -1px;
  background: var(--vc-line);
}
.invite-actions {
  display: flex;
  flex-direction: row;
  align-items: stretch;
  gap: 12px;
  padding-top: 4px;
  border-top: 1px solid var(--vc-line);
}
.invite-action {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.invite-btn {
  border-radius: 10px !important;
  height: auto !important;
  min-height: 36px;
  padding: 6px 10px !important;
  white-space: nowrap;
}
.invite-btn-guest {
  border-color: #f3a04c !important;
  background: #f3a04c !important;
  color: #fff !important;
  box-shadow: 0 8px 18px rgba(243, 160, 76, 0.28) !important;
}
.invite-btn-guest:hover {
  border-color: #e8892a !important;
  background: #e8892a !important;
  color: #fff !important;
}
.invite-btn-member {
  border-color: #3b82f6 !important;
  background: #3b82f6 !important;
  color: #fff !important;
  box-shadow: 0 8px 18px rgba(59, 130, 246, 0.28) !important;
}
.invite-btn-member:hover {
  border-color: #2563eb !important;
  background: #2563eb !important;
  color: #fff !important;
}
.invite-hint {
  margin: 0;
  padding: 0 2px;
  color: var(--vc-muted);
  font-size: 12px;
  line-height: 1.45;
}
.tabular {
  font-variant-numeric: tabular-nums;
}
@media (max-width: 800px) {
  .stage.speaker {
    grid-template-columns: 1fr;
  }
  .speaker-side {
    flex-direction: row;
    max-height: 140px;
  }
  .tile.side {
    width: 160px;
  }
  .top,
  .controls {
    margin-left: 10px;
    margin-right: 10px;
  }
}
@media (max-width: 520px) {
  .invite-actions {
    flex-direction: column;
  }
}
</style>

<style>
.room-invite-modal .ant-modal-content {
  overflow: hidden;
  border-radius: 14px;
}
.room-invite-modal .ant-modal-header {
  margin: 0;
  padding: 20px 24px 8px;
  border-bottom: none;
}
.room-invite-modal .ant-modal-title {
  color: var(--vc-ink, rgba(15, 23, 42, 0.92));
  font-size: 17px;
  font-weight: 650;
  line-height: 1.35;
}
.room-invite-modal .ant-modal-close {
  top: 16px;
}
.room-invite-modal .ant-modal-body {
  padding: 4px 24px 22px;
}
</style>
