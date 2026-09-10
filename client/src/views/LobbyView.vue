<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Button,
  DatePicker,
  Dropdown,
  Empty,
  Form,
  Input,
  Modal,
  Select,
  Switch,
  message,
} from 'ant-design-vue'
import {
  CalendarOutlined,
  CheckOutlined,
  ClockCircleOutlined,
  CloseOutlined,
  CopyOutlined,
  EditOutlined,
  HistoryOutlined,
  InfoCircleOutlined,
  LogoutOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  UserAddOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons-vue'
import dayjs, { type Dayjs } from 'dayjs'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { fetchMe, logout as apiLogout } from '@/api/auth'
import {
  createMeeting,
  createToken,
  deleteMeeting,
  endMeeting,
  listMeetings,
  listMeetingTypes,
  updateMeeting,
  downloadRecordingFile,
  recordingPlaySrc,
  recordingStatus,
  viewMinutes,
  regenerateMinutes,
  type MeetingItem,
  type MeetingTypeOption,
  type MinutesInfo,
  type RecordingSegment,
} from '@/api/conference'
import { clearAuth, displayName, getAuth, setAuth, subscribeAuth } from '@/stores/auth'
import { ApiError } from '@/utils/request'
import { writeMeetingSession } from '@/utils/meetingSession'

type FilterKey = 'all' | 'ongoing' | 'host' | 'joined'

/** 未分类筛选项的伪 ID（与后端 type_id=0 对应） */
const NO_TYPE_ID = 0

const router = useRouter()
const loading = ref(false)
const creating = ref(false)
const createOpen = ref(false)
const renaming = ref(false)
const editOpen = ref(false)
const inviteOpen = ref(false)
const inviteTarget = ref<MeetingItem | null>(null)
const detailOpen = ref(false)
const detailMeeting = ref<MeetingItem | null>(null)
const detailMinutes = ref<MinutesInfo | null>(null)
const minutesLoading = ref(false)
const minutesRegenning = ref(false)
const minutesWatchId = ref<number | null>(null)
const minutesHintLoading = ref(false)
const minutesHintText = ref('正在整理会议纪要…')
let minutesSettleToken = 0
const playSeg = ref<RecordingSegment | null>(null)
const playError = ref('')
const meetings = ref<MeetingItem[]>([])
const userLabel = ref(displayName())
const filter = ref<FilterKey>('all')
/** 会议类型选项（管理端维护） */
const meetingTypes = ref<MeetingTypeOption[]>([])
/** 列表标题处多选的类型筛选（空=全部） */
const selectedTypeIdsRaw = ref<number[]>([])
const selectedTypeIds = computed<number[]>({
  get: () => selectedTypeIdsRaw.value,
  set: (value) => {
    // 多选清空时组件可能回传 undefined，统一归一为空数组
    selectedTypeIdsRaw.value = Array.isArray(value) ? value : []
  },
})
/** 类型下拉展开态（用于箭头旋转） */
const typeFilterOpen = ref(false)
/** 顶栏搜索：展开态与关键词（按会议名称模糊匹配） */
const searchOpen = ref(false)
const searchKeyword = ref('')
const searchInputRef = ref<{ focus?: () => void } | null>(null)

type InviteKind = 'guest' | 'member'

const inviteTime = computed(() =>
  inviteTarget.value ? formatInviteTime(inviteTarget.value) : null,
)

const createForm = reactive({
  title: '',
  startAt: undefined as Dayjs | undefined,
  endAt: undefined as Dayjs | undefined,
  recordEnabled: false,
  typeId: undefined as number | undefined,
})

const editForm = reactive({
  id: 0,
  title: '',
  startAt: undefined as Dayjs | undefined,
  endAt: undefined as Dayjs | undefined,
  typeId: undefined as number | undefined,
})

const sortedMeetings = computed(() => {
  const list = [...meetings.value]
  list.sort((a, b) => dayjs(b.startAt).valueOf() - dayjs(a.startAt).valueOf())
  return list
})

/** 列表标题处类型多选选项：管理端维护的类型 + 「未分类」 */
const typeFilterOptions = computed(() => [
  { value: NO_TYPE_ID, label: '未分类' },
  ...meetingTypes.value.map((t) => ({ value: t.id, label: t.name })),
])

/** 新建/编辑会议弹窗的类型选项（不含「未分类」，不选即为未分类） */
const typeFormOptions = computed(() =>
  meetingTypes.value.map((t) => ({ value: t.id, label: t.name })),
)

/** 类型配色数量（与 style 中的 .type-c0 ~ .type-c9 对应） */
const TYPE_COLOR_COUNT = 10

/**
 * 会议类型配色类名：同一类型在筛选标签、下拉项、列表标签上颜色一致。
 * 按类型ID取模，保证相邻类型颜色不同；0（未分类）用中性灰。
 */
function typeClass(typeId?: number) {
  const id = Number(typeId ?? NO_TYPE_ID)
  if (!Number.isFinite(id) || id <= 0) return 'type-none'
  return `type-c${Math.abs(id) % TYPE_COLOR_COUNT}`
}

/** 已选类型（保持与选项一致的顺序） */
const selectedTypeTags = computed(() =>
  typeFilterOptions.value.filter((o) => selectedTypeIds.value.includes(o.value)),
)

/** 标题行最多展示的已选标签数，超出用 +N 收起 */
const TYPE_TAG_LIMIT = 3
const visibleTypeTags = computed(() => selectedTypeTags.value.slice(0, TYPE_TAG_LIMIT))
const hiddenTypeTagCount = computed(() =>
  Math.max(0, selectedTypeTags.value.length - TYPE_TAG_LIMIT),
)

/** 点选/取消一个会议类型（多选，不关闭面板） */
function toggleTypeFilter(typeId: number) {
  const picked = new Set(selectedTypeIds.value)
  if (picked.has(typeId)) {
    picked.delete(typeId)
  } else {
    picked.add(typeId)
  }
  selectedTypeIds.value = typeFilterOptions.value
    .filter((o) => picked.has(o.value))
    .map((o) => o.value)
}

/** 清除全部类型筛选 */
function clearTypeFilter() {
  selectedTypeIds.value = []
}

/** 搜索关键词（去空格、忽略大小写） */
const keyword = computed(() => searchKeyword.value.trim().toLowerCase())

/** 是否处于筛选状态（类型多选或关键词搜索） */
const hasExtraFilter = computed(() => selectedTypeIds.value.length > 0 || !!keyword.value)

function myNameCandidates() {
  const auth = getAuth()
  return [displayName(), auth?.realName, auth?.username]
    .map((n) => n?.trim())
    .filter((n): n is string => !!n)
}

/** 参会名单里出现过当前用户显示名 / 用户名（进房写 attendees） */
function didJoin(m: MeetingItem) {
  const names = attendeesOf(m)
  if (!names.length) return false
  const mine = new Set(myNameCandidates())
  return names.some((n) => mine.has(n))
}

const filteredMeetings = computed(() => {
  let list = sortedMeetings.value
  if (filter.value === 'ongoing') {
    list = list.filter((m) => m.tab === 'ongoing')
  } else if (filter.value === 'host') {
    list = list.filter((m) => m.isHost)
  } else if (filter.value === 'joined') {
    list = list.filter((m) => didJoin(m))
  }
  // 类型多选（含「未分类」）；未选择时不筛选
  if (selectedTypeIds.value.length) {
    const picked = new Set(selectedTypeIds.value)
    list = list.filter((m) => picked.has(m.typeId ?? NO_TYPE_ID))
  }
  // 按会议名称模糊搜索
  if (keyword.value) {
    list = list.filter((m) => (m.title || '').toLowerCase().includes(keyword.value))
  }
  return list
})

const listTitle = computed(() => {
  switch (filter.value) {
    case 'ongoing':
      return '进行中'
    case 'host':
      return '我主持'
    case 'joined':
      return '我参与'
    default:
      return '全部会议'
  }
})

const emptyDescription = computed(() => {
  if (keyword.value) return `没有名称包含「${searchKeyword.value.trim()}」的会议`
  if (selectedTypeIds.value.length) return '没有符合所选类型的会议'
  switch (filter.value) {
    case 'ongoing':
      return '当前没有进行中的会议'
    case 'host':
      return '还没有你主持的会议'
    case 'joined':
      return '还没有你参与过的会议'
    default:
      return '还没有会议，创建一个开始协作吧'
  }
})

const historyCount = computed(() => meetings.value.filter((m) => m.tab === 'ended').length)
const ongoingCount = computed(() => meetings.value.filter((m) => m.tab === 'ongoing').length)
const scheduledCount = computed(() => meetings.value.filter((m) => m.tab === 'scheduled').length)
const todayCount = computed(() => {
  const today = dayjs().format('YYYY-MM-DD')
  return meetings.value.filter((m) => dayjs(m.startAt).format('YYYY-MM-DD') === today).length
})

let unsubAuth: (() => void) | undefined

function syncUser() {
  userLabel.value = displayName()
}

async function refresh(opts?: { silent?: boolean }) {
  if (!opts?.silent) loading.value = true
  try {
    const data = await listMeetings('all')
    meetings.value = data.list ?? []
    syncDetailMinutesFromList()
    notifyWatchedMinutesIfSettled()
    if (anyMinutesBusy() || minutesWatchId.value != null) startMinutesListPoll()
    else stopMinutesListPoll()
  } catch (err) {
    if (!opts?.silent) {
      message.error(err instanceof ApiError ? err.message : '加载会议列表失败')
    }
  } finally {
    if (!opts?.silent) loading.value = false
  }
}

/** 加载管理端维护的会议类型；失败不阻塞列表，仅降级为无类型筛选 */
async function loadMeetingTypes() {
  try {
    const data = await listMeetingTypes()
    meetingTypes.value = data.list ?? []
    // 类型被删除后清理已选项，避免出现「已选但筛不出」的僵状态
    if (selectedTypeIds.value.length) {
      const valid = new Set<number>([NO_TYPE_ID, ...meetingTypes.value.map((t) => t.id)])
      selectedTypeIds.value = selectedTypeIds.value.filter((id) => valid.has(id))
    }
  } catch {
    meetingTypes.value = []
  }
}

function openSearch() {
  searchOpen.value = true
  void nextTick(() => {
    searchInputRef.value?.focus?.()
  })
}

/** 关闭搜索框并清空关键词，恢复完整列表 */
function closeSearch() {
  searchOpen.value = false
  searchKeyword.value = ''
}

async function ensureMe() {
  try {
    const me = await fetchMe()
    const auth = getAuth()
    if (auth) {
      setAuth({ ...auth, username: me.username, realName: me.realName, id: me.id })
      syncUser()
    }
  } catch {
    // ignore
  }
}

function shareLink(m: MeetingItem) {
  return `${window.location.origin}/join/${m.shareCode}`
}

function openInvite(m: MeetingItem) {
  inviteTarget.value = m
  inviteOpen.value = true
}

function clearInviteTarget() {
  inviteTarget.value = null
}

function attendeesOf(m: MeetingItem) {
  return (m.attendees ?? []).filter((n) => {
    const name = n?.trim()
    return !!name && !name.startsWith('EG_')
  })
}

function attendeesPreview(m: MeetingItem) {
  const list = attendeesOf(m)
  if (!list.length) return '暂无'
  if (list.length <= 6) return list.join('、')
  return `${list.slice(0, 6).join('、')} 等 ${list.length} 人`
}

function recordingsOf(m: MeetingItem) {
  return (m.recordings ?? []).filter((seg) => !seg.purpose || seg.purpose === 'playback')
}

function minutesStatusText(status?: string) {
  switch (status) {
    case 'pending':
    case 'transcribing':
    case 'summarizing':
    case undefined:
    case '':
      return '编写中'
    case 'ready':
      return '已生成'
    case 'failed':
      return '生成失败'
    case 'skipped_empty':
    case 'unavailable':
      return '无法生成'
    default:
      // 会议刚结束、纪要行尚未落库时也视为编写中
      return status || '编写中'
  }
}

function isMinutesBusy(status?: string) {
  return !status || status === 'pending' || status === 'transcribing' || status === 'summarizing'
}

function isMinutesNoAudio(status?: string) {
  return status === 'unavailable' || status === 'skipped_empty'
}

function minutesOf(m: MeetingItem): MinutesInfo | null {
  return m.minutes || null
}

let minutesListPollTimer: ReturnType<typeof setInterval> | null = null

function anyMinutesBusy() {
  return meetings.value.some((item) => isEnded(item) && isMinutesBusy(minutesOf(item)?.status))
}

function startMinutesListPoll() {
  if (minutesListPollTimer) return
  minutesListPollTimer = setInterval(() => {
    void refresh({ silent: true })
  }, 3000)
}

function stopMinutesListPoll() {
  if (!minutesListPollTimer) return
  clearInterval(minutesListPollTimer)
  minutesListPollTimer = null
}

function syncDetailMinutesFromList() {
  const current = detailMeeting.value
  if (!current) return
  const fresh = meetings.value.find((item) => item.id === current.id)
  if (!fresh) return
  if (fresh.minutes) {
    detailMinutes.value = fresh.minutes
    current.minutes = fresh.minutes
  }
}

function notifyWatchedMinutesIfSettled() {
  const id = minutesWatchId.value
  if (id == null) return
  const item = meetings.value.find((m) => m.id === id)
  const st = item?.minutes?.status
  if (isMinutesBusy(st)) return
  minutesWatchId.value = null
  if (st === 'ready') {
    message.success('会议纪要已生成')
    return
  }
  if (isMinutesNoAudio(st)) {
    message.warning('该会议没有音频，无法生成会议纪要')
    return
  }
  if (st === 'failed') {
    message.error('会议纪要生成失败')
  }
}

function minutesHintSettled(info?: MinutesInfo | null) {
  const status = info?.status
  if (!status) return false
  if (isMinutesNoAudio(status) || status === 'failed' || status === 'ready') return true
  if (status === 'transcribing' || status === 'summarizing') return true
  if (status === 'pending' && (info?.sourceRecordingIds?.length ?? 0) > 0) return true
  return false
}

function waitMs(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms))
}

async function settleMinutesAfterEnd(meetingId: number, initial?: MinutesInfo) {
  const token = ++minutesSettleToken
  minutesHintLoading.value = true
  minutesHintText.value = '正在整理会议纪要…'
  let last = initial
  const deadline = Date.now() + 15000
  try {
    if (minutesHintSettled(last)) {
      if (token !== minutesSettleToken) return
      minutesHintLoading.value = false
      showMinutesEndHint(meetingId, last)
      return
    }
    while (Date.now() < deadline) {
      if (token !== minutesSettleToken) return
      try {
        last = await viewMinutes(meetingId)
      } catch {
        // 音源收尾期间允许短暂查不到
      }
      if (token !== minutesSettleToken) return
      if (minutesHintSettled(last)) break
      await waitMs(700)
    }
    if (token !== minutesSettleToken) return
    minutesHintLoading.value = false
    showMinutesEndHint(
      meetingId,
      last?.status ? last : { meetingId, status: 'pending' },
    )
    await refresh({ silent: true })
  } catch {
    if (token !== minutesSettleToken) return
    minutesHintLoading.value = false
    showMinutesEndHint(meetingId, last)
  }
}

function showMinutesEndHint(meetingId: number, info?: MinutesInfo) {
  if (!info) {
    Modal.info({
      title: '会议已结束',
      content: '会议已结束。',
      okText: '知道了',
    })
    return
  }
  const status = info.status
  if (isMinutesNoAudio(status)) {
    Modal.info({
      title: '会议已结束',
      content: '该会议没有音频，无法生成会议纪要。',
      okText: '知道了',
    })
    return
  }
  if (isMinutesBusy(status) || !status) {
    minutesWatchId.value = meetingId
    Modal.info({
      title: '会议已结束',
      content: '会议纪要正在编写，请稍候，结果出来后页面会自动更新。',
      okText: '知道了',
    })
    startMinutesListPoll()
    return
  }
  if (status === 'ready') {
    Modal.info({
      title: '会议已结束',
      content: '会议纪要已生成，可在会议详情中查看。',
      okText: '知道了',
    })
    return
  }
  Modal.info({
    title: '会议已结束',
    content: status === 'failed' ? '会议纪要生成失败。' : '会议已结束。',
    okText: '知道了',
  })
}

function canPlaySeg(seg: RecordingSegment) {
  return seg.status === 'complete' && !!seg.id
}

function recordingStatusText(seg: RecordingSegment) {
  switch (seg.status) {
    case 'complete':
      return canPlaySeg(seg) ? '' : '文件未就绪'
    case 'failed':
      return '失败'
    case 'starting':
    case 'active':
    case 'stopping':
      return '处理中'
    default:
      return seg.status || '未知'
  }
}

function isProcessingSeg(seg: RecordingSegment) {
  return ['starting', 'active', 'stopping'].includes(seg.status)
}

const playSrc = computed(() => {
  if (!playSeg.value || !canPlaySeg(playSeg.value)) return ''
  return recordingPlaySrc(playSeg.value.id)
})

function openDetail(m: MeetingItem, seg?: RecordingSegment) {
  detailMeeting.value = m
  detailMinutes.value = minutesOf(m)
  playError.value = ''
  playSeg.value = seg && canPlaySeg(seg) ? seg : firstPlayableSeg(m)
  detailOpen.value = true
  void refreshDetailRecordings()
  void refreshDetailMinutes()
  startDetailPoll()
}

function closeDetail() {
  // 关动画未结束又点开时，afterClose 会晚到；已重新打开则不要清掉新状态
  if (detailOpen.value) return
  stopDetailPoll()
  detailMeeting.value = null
  detailMinutes.value = null
  playSeg.value = null
  playError.value = ''
}

function firstPlayableSeg(m: MeetingItem) {
  return recordingsOf(m).find((s) => canPlaySeg(s)) ?? null
}

function selectPlaySeg(seg: RecordingSegment) {
  if (!canPlaySeg(seg)) return
  playError.value = ''
  playSeg.value = seg
}

function onPlayError() {
  playError.value = '无法播放该录制，请稍后重试或改用下载'
}

let detailPollTimer: ReturnType<typeof setInterval> | null = null

async function refreshDetailMinutes() {
  const m = detailMeeting.value
  if (!m) return
  minutesLoading.value = true
  try {
    const info = await viewMinutes(m.id)
    detailMinutes.value = info
    m.minutes = info
  } catch (e) {
    // 列表里可能已有摘要；详情拉取失败不打断回放
    console.warn('refresh minutes failed', e)
  } finally {
    minutesLoading.value = false
  }
}

async function onRegenerateMinutes() {
  const m = detailMeeting.value
  if (!m || !m.isHost) return
  minutesRegenning.value = true
  try {
    const info = await regenerateMinutes(m.id)
    detailMinutes.value = info
    m.minutes = info
    message.success('已重新提交纪要生成')
    void refreshDetailMinutes()
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : '重新生成失败'
    message.error(msg)
  } finally {
    minutesRegenning.value = false
  }
}

function startDetailPoll() {
  stopDetailPoll()
  detailPollTimer = setInterval(() => {
    const m = detailMeeting.value
    if (!m) return
    const mins = detailMinutes.value
    const minutesBusy =
      mins &&
      (mins.status === 'pending' ||
        mins.status === 'transcribing' ||
        mins.status === 'summarizing')
    if (!recordingsOf(m).some(isProcessingSeg) && !minutesBusy) {
      stopDetailPoll()
      return
    }
    void refreshDetailRecordings()
    if (minutesBusy) void refreshDetailMinutes()
  }, 3000)
}

function stopDetailPoll() {
  if (detailPollTimer) {
    clearInterval(detailPollTimer)
    detailPollTimer = null
  }
}

async function refreshDetailRecordings() {
  const m = detailMeeting.value
  if (!m?.id) return
  try {
    const res = await recordingStatus({ meetingId: m.id })
    if (detailMeeting.value?.id !== m.id) return
    const segments = res.segments ?? []
    detailMeeting.value = { ...m, recordings: segments }
    const idx = meetings.value.findIndex((item) => item.id === m.id)
    if (idx >= 0) {
      meetings.value[idx] = { ...meetings.value[idx], recordings: segments }
    }
    if (playSeg.value) {
      const latest = segments.find((s) => s.id === playSeg.value?.id)
      if (latest) playSeg.value = latest
    } else {
      playSeg.value = segments.find((s) => canPlaySeg(s)) ?? null
    }
  } catch {
    // 详情刷新失败不打断已打开的播放
  }
}

function formatFileSize(n?: number) {
  if (!n || n <= 0) return ''
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function formatSegSpan(seg: RecordingSegment) {
  if (!seg.startedAt) return ''
  const start = dayjs(seg.startedAt)
  if (!seg.endedAt) return start.format('HH:mm')
  const end = dayjs(seg.endedAt)
  return `${start.format('HH:mm')} – ${end.format('HH:mm')}`
}

function recordingDownloadName(m: MeetingItem, seg: RecordingSegment) {
  const title = (m.title || '会议录制').replace(/[\\/:*?"<>|]/g, '_')
  return `${title}-第${seg.seq}段.mp4`
}

async function downloadSeg(m: MeetingItem, seg: RecordingSegment) {
  if (!seg.id) return
  const hide = message.loading('正在下载…', 0)
  try {
    await downloadRecordingFile(seg.id, recordingDownloadName(m, seg))
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '下载失败')
  } finally {
    hide()
  }
}

function buildInviteText(m: MeetingItem, kind: InviteKind) {
  const how =
    kind === 'guest'
      ? '打开下方链接，填写昵称即可进入（无需账号）'
      : '请使用公司账号登录后，通过下方链接进入会议'
  return [
    '【视频会议邀请】',
    `主题：${m.title || '-'}`,
    `主持人：${m.hostName || '-'}`,
    `时间：${formatInviteTimeText(m)}`,
    `加入方式：${how}`,
    `会议链接：${shareLink(m)}`,
  ].join('\n')
}

async function copyInvite(kind: InviteKind) {
  const m = inviteTarget.value
  if (!m) return
  const text = buildInviteText(m, kind)
  try {
    await navigator.clipboard.writeText(text)
    message.success(kind === 'guest' ? '游客邀请已复制' : '同事邀请已复制')
  } catch {
    message.info(text)
  }
}

function canEnter(m: MeetingItem) {
  if (m.tab === 'ended') return false
  if (m.tab === 'ongoing') return true
  if (!m.startAt) return true
  // 可提前 5 分钟进入
  return !dayjs(m.startAt).subtract(5, 'minute').isAfter(dayjs())
}

/** 主持人：仅「进行中」的会议可结束 */
function canEnd(m: MeetingItem) {
  return m.isHost && isOngoing(m)
}

/** 主持人：仅「预定」的会议可删除（取消并入删除）；已结束的会议不可在会议端删 */
function canDelete(m: MeetingItem) {
  return m.isHost && !isOngoing(m) && !isEnded(m)
}

async function enterMeeting(m: MeetingItem) {
  if (m.tab === 'ended') {
    message.info('会议已结束')
    return
  }
  if (!canEnter(m)) {
    message.info('会议尚未开始')
    return
  }
  const nick = displayName() || '用户'
  try {
    const data = await createToken({ room: m.roomName, nickname: nick })
    writeMeetingSession({
      serverUrl: data.serverUrl,
      token: data.token,
      room: data.room,
      title: data.title || m.title,
      identity: data.identity,
      nickname: data.nickname,
      expiresAt: data.expiresAt,
      isHost: !!data.isHost,
      enableMic: false,
      enableCamera: false,
      fromShare: false,
      shareCode: m.shareCode,
      recordEnabled: !!data.recordEnabled,
      recordingActive: !!data.recordingActive,
      startAt: data.startAt || m.startAt,
      actualStartAt: data.actualStartAt || m.actualStartAt,
    })
    await router.push({ name: 'room', params: { room: data.room } })
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '进入会议失败')
  }
}

async function onEnd(m: MeetingItem) {
  Modal.confirm({
    title: '结束会议',
    content: `确定结束「${m.title}」？结束后将计入历史会议，无法再加入。`,
    okText: '结束',
    cancelText: '取消',
    okType: 'danger',
    async onOk() {
      try {
        minutesHintLoading.value = true
        minutesHintText.value = '正在整理会议纪要…'
        const data = await endMeeting(m.id)
        await refresh({ silent: true })
        void settleMinutesAfterEnd(m.id, data?.minutes)
      } catch (err) {
        minutesHintLoading.value = false
        message.error(err instanceof ApiError ? err.message : '操作失败')
        throw err
      }
    },
  })
}

async function onDelete(m: MeetingItem) {
  Modal.confirm({
    title: '删除会议',
    content: `确定删除「${m.title}」？删除后不可恢复，适用于建错后重建。`,
    okText: '删除',
    cancelText: '取消',
    okType: 'danger',
    async onOk() {
      try {
        await deleteMeeting(m.id)
        message.success('已删除')
        await refresh()
      } catch (err) {
        message.error(err instanceof ApiError ? err.message : '删除失败')
        throw err
      }
    },
  })
}

function canEdit(m: MeetingItem) {
  return m.isHost && !isOngoing(m) && !isEnded(m)
}

function openEdit(m: MeetingItem) {
  if (!canEdit(m)) {
    message.info('进行中或已结束的会议不可修改')
    return
  }
  editForm.id = m.id
  editForm.title = m.title
  editForm.startAt = m.startAt ? dayjs(m.startAt) : undefined
  editForm.endAt = m.endAt ? dayjs(m.endAt) : undefined
  editForm.typeId = m.typeId && m.typeId > 0 ? m.typeId : undefined
  editOpen.value = true
}

async function onEdit() {
  const title = editForm.title.trim()
  if (!title) {
    message.warning('请填写会议名称')
    return Promise.reject()
  }
  if ([...title].length > 64) {
    message.warning('会议名称最长 64 个字符')
    return Promise.reject()
  }
  if (!editForm.startAt || !editForm.endAt) {
    message.warning('请选择会议开始与结束时间')
    return Promise.reject()
  }
  if (!editForm.endAt.isAfter(editForm.startAt)) {
    message.warning('结束时间必须晚于开始时间')
    return Promise.reject()
  }
  if (editForm.startAt.isBefore(dayjs())) {
    message.warning('开始时间不能早于当前时间')
    return Promise.reject()
  }
  renaming.value = true
  try {
    await updateMeeting({
      id: editForm.id,
      title,
      startAt: editForm.startAt.format('YYYY-MM-DD HH:mm:ss'),
      endAt: editForm.endAt.format('YYYY-MM-DD HH:mm:ss'),
      typeId: editForm.typeId ?? NO_TYPE_ID,
    })
    message.success('会议已更新')
    editOpen.value = false
    await refresh()
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '更新失败')
    return Promise.reject(err)
  } finally {
    renaming.value = false
  }
}

async function onCreate() {
  if (!createForm.title.trim()) {
    message.warning('请填写会议名称')
    return
  }
  if (!createForm.startAt || !createForm.endAt) {
    message.warning('请选择会议开始与结束时间')
    return
  }
  if (!createForm.endAt.isAfter(createForm.startAt)) {
    message.warning('结束时间必须晚于开始时间')
    return
  }
  if (createForm.startAt.isBefore(dayjs())) {
    message.warning('开始时间不能早于当前时间')
    return
  }
  creating.value = true
  try {
    await createMeeting({
      title: createForm.title.trim(),
      hostName: displayName(),
      startAt: createForm.startAt.format('YYYY-MM-DD HH:mm:ss'),
      endAt: createForm.endAt.format('YYYY-MM-DD HH:mm:ss'),
      recordEnabled: !!createForm.recordEnabled,
      typeId: createForm.typeId ?? NO_TYPE_ID,
    })
    message.success('会议室已创建')
    createOpen.value = false
    resetCreateForm()
    await refresh()
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '创建失败')
  } finally {
    creating.value = false
  }
}

// 生成 [start, end) 的整数数组
function range(start: number, end: number): number[] {
  return Array.from({ length: Math.max(0, end - start) }, (_, i) => start + i)
}

// 新建会议：开始日期选「今天」时，禁用当前时刻之前的时间（日期维度的禁用由 disabled-date 处理）
function disabledCreateStartTime(current: Dayjs | null) {
  const now = dayjs()
  if (!current || !current.isSame(now, 'day')) {
    return {}
  }
  const disabledHours = () => range(0, now.hour())
  const disabledMinutes = (hour: number) => (hour === now.hour() ? range(0, now.minute()) : [])
  const disabledSeconds = () => range(0, now.second())
  return { disabledHours, disabledMinutes, disabledSeconds }
}

function resetCreateForm() {
  createForm.title = ''
  createForm.recordEnabled = false
  createForm.typeId = undefined
  const now = dayjs()
  // 默认从现在起 5 分钟开始、整 1 小时结束：
  // - startAt 取整到分钟，避免因带秒数导致“到点却不开始/开始时间晚于表单显示时间”
  // - endAt 由 startAt 推导，保证默认时长固定为 1 小时而不是 55 分钟
  createForm.startAt = now.add(5, 'minute').startOf('minute')
  createForm.endAt = createForm.startAt.add(1, 'hour')
}

function openCreate() {
  resetCreateForm()
  createOpen.value = true
}

async function onLogout() {
  try {
    await apiLogout()
  } catch {
    // ignore
  }
  clearAuth()
  await router.replace({ name: 'login' })
}

function formatDurationMinutes(mins: number) {
  if (mins <= 0) return '不足 1 分钟'
  if (mins >= 60) {
    const h = Math.floor(mins / 60)
    const rem = mins % 60
    return rem ? `${h} 小时 ${rem} 分钟` : `${h} 小时`
  }
  return `${mins} 分钟`
}

/** 2026 年 8 月 12 日：数字与单位留空隙，月份不补零 */
function formatCnDate(d: dayjs.Dayjs) {
  return `${d.year()} 年 ${d.month() + 1} 月 ${d.date()} 日`
}

function formatMeetingSchedule(m: MeetingItem) {
  if (!m.startAt) {
    return { date: '-', range: '-', duration: '' }
  }
  const start = dayjs(m.startAt)
  const end = m.endAt ? dayjs(m.endAt) : null
  const date = formatCnDate(start)
  let range = start.format('HH:mm')
  if (end) {
    range = end.isSame(start, 'day')
      ? `${start.format('HH:mm')} – ${end.format('HH:mm')}`
      : `${start.format('HH:mm')} – ${formatCnDate(end)} ${end.format('HH:mm')}`
  }
  const duration = end ? formatDurationMinutes(end.diff(start, 'minute')) : ''
  return { date, range, duration }
}

function formatInviteTime(m: MeetingItem) {
  if (!m.startAt) {
    return { date: '-', range: '-', duration: '' }
  }
  const start = dayjs(m.startAt)
  const end = m.endAt ? dayjs(m.endAt) : null
  const weekdays = ['日', '一', '二', '三', '四', '五', '六']
  const date = `${formatCnDate(start)} 周${weekdays[start.day()]}`
  const range = end
    ? end.isSame(start, 'day')
      ? `${start.format('HH:mm')} – ${end.format('HH:mm')}`
      : `${start.format('HH:mm')} – ${formatCnDate(end)} ${end.format('HH:mm')}`
    : start.format('HH:mm')
  const duration = end ? formatDurationMinutes(end.diff(start, 'minute')) : ''
  return { date, range, duration }
}

function formatInviteTimeText(m: MeetingItem) {
  const t = formatInviteTime(m)
  const label = isEnded(m) ? '实际时长' : '预计时长'
  if (!t.duration) return `${t.date} ${t.range}`
  return `${t.date} ${t.range}，${label} ${t.duration}`
}

function isOngoing(m: MeetingItem) {
  return m.tab === 'ongoing'
}

function isEnded(m: MeetingItem) {
  return m.tab === 'ended'
}

function statusLabel(m: MeetingItem) {
  if (m.tab === 'ongoing') return '进行中'
  if (m.tab === 'ended') return '已结束'
  return '预定'
}

function progressInfo(m: MeetingItem) {
  if (!m.startAt || !m.endAt) return null
  const start = m.actualStartAt ? dayjs(m.actualStartAt) : dayjs(m.startAt)
  const end = dayjs(m.endAt)
  const now = dayjs()
  const total = Math.max(end.diff(start, 'minute'), 1)
  const elapsed = Math.min(Math.max(now.diff(start, 'minute'), 0), total)
  const percent = Math.min(100, Math.round((elapsed / total) * 100))
  return { elapsed, percent }
}

onMounted(() => {
  unsubAuth = subscribeAuth(syncUser)
  void ensureMe()
  void loadMeetingTypes()
  void refresh()
  resetCreateForm()
})

onUnmounted(() => {
  unsubAuth?.()
  stopDetailPoll()
  stopMinutesListPoll()
  minutesSettleToken += 1
  minutesHintLoading.value = false
})
</script>

<template>
  <div class="lobby">
    <div class="blob blob-a" aria-hidden="true" />
    <div class="blob blob-b" aria-hidden="true" />
    <div class="blob blob-c" aria-hidden="true" />

    <div class="shell">
      <!-- 顶栏 -->
      <header class="nav">
        <div class="nav-left">
          <img class="logo" src="/favicon.png" alt="" width="42" height="42" aria-hidden="true" />
          <div class="nav-titles">
            <h1>会议大厅</h1>
            <p class="nav-sub">
              <span class="user">
                {{ userLabel || '用户' }}
              </span>
              <span class="divider">·</span>
              <span class="nums">
                <em>{{ ongoingCount }}</em> 场进行中
                <span class="slash">·</span>
                <em class="soft">{{ scheduledCount }}</em> 场预定
              </span>
            </p>
          </div>
        </div>
        <div class="nav-right">
          <div v-if="searchOpen" class="search-box">
            <Input
              ref="searchInputRef"
              v-model:value="searchKeyword"
              class="search-input"
              placeholder="搜索会议名称"
              allow-clear
            >
              <template #prefix><SearchOutlined class="search-prefix" /></template>
            </Input>
            <Button
              type="text"
              class="icon-btn"
              title="关闭搜索"
              aria-label="关闭搜索"
              @click="closeSearch"
            >
              <template #icon><CloseOutlined /></template>
            </Button>
          </div>
          <Button
            v-else
            type="text"
            class="icon-btn"
            title="搜索会议"
            aria-label="搜索会议"
            @click="openSearch"
          >
            <template #icon><SearchOutlined /></template>
          </Button>
          <ThemeToggle />
          <Button type="text" class="icon-btn" :loading="loading" @click="() => refresh()">
            <template #icon><ReloadOutlined /></template>
          </Button>
          <Button type="primary" class="btn-primary" @click="openCreate">
            <template #icon><PlusOutlined /></template>
            新建会议室
          </Button>
          <Button class="btn-secondary" @click="onLogout">
            <template #icon><LogoutOutlined /></template>
            退出
          </Button>
        </div>
      </header>

      <!-- 数据概览 -->
      <section class="stats" aria-label="数据概览">
        <article class="stat-card" style="--delay: 0ms">
          <div class="stat-icon green"><VideoCameraOutlined /></div>
          <div class="stat-body">
            <div class="stat-num">{{ ongoingCount }}</div>
            <div class="stat-label">进行中会议</div>
          </div>
        </article>
        <article class="stat-card" style="--delay: 70ms">
          <div class="stat-icon blue"><CalendarOutlined /></div>
          <div class="stat-body">
            <div class="stat-num">{{ scheduledCount }}</div>
            <div class="stat-label">预定会议</div>
          </div>
        </article>
        <article class="stat-card" style="--delay: 140ms">
          <div class="stat-icon amber"><ClockCircleOutlined /></div>
          <div class="stat-body">
            <div class="stat-num">{{ todayCount }}</div>
            <div class="stat-label">今日会议</div>
          </div>
        </article>
        <article class="stat-card" style="--delay: 210ms">
          <div class="stat-icon violet"><HistoryOutlined /></div>
          <div class="stat-body">
            <div class="stat-num">{{ historyCount }}</div>
            <div class="stat-label">历史会议</div>
          </div>
        </article>
      </section>

      <!-- 会议列表 -->
      <section class="list-panel">
        <div class="list-head">
          <div class="list-title">
            <h2>{{ listTitle }}</h2>
            <span class="badge">{{ filteredMeetings.length }}</span>
            <!-- 折叠按钮：点击展开会议类型多选面板（选项来自管理端「会议类型」） -->
            <Dropdown
              v-if="typeFilterOptions.length > 1"
              v-model:open="typeFilterOpen"
              trigger="click"
              placement="bottomLeft"
              overlay-class-name="type-panel-dropdown"
            >
              <button
                type="button"
                class="type-filter-btn"
                :class="{ active: selectedTypeIds.length > 0, open: typeFilterOpen }"
                :title="
                  selectedTypeIds.length
                    ? `已筛选 ${selectedTypeIds.length} 个会议类型`
                    : '按会议类型筛选'
                "
                aria-label="按会议类型筛选"
              >
                <svg
                  class="type-filter-caret"
                  viewBox="0 0 16 16"
                  width="13"
                  height="13"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.7"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M4 6.5 8 10.5 12 6.5" />
                </svg>
                <span v-if="selectedTypeIds.length" class="type-filter-dot" aria-hidden="true" />
              </button>
              <template #overlay>
                <div class="type-panel">
                  <div class="type-panel-head">
                    <span class="type-panel-title">会议类型</span>
                    <button
                      v-if="selectedTypeIds.length"
                      type="button"
                      class="type-panel-clear"
                      @click="clearTypeFilter"
                    >
                      清除
                    </button>
                  </div>
                  <div class="type-panel-list">
                    <button
                      v-for="opt in typeFilterOptions"
                      :key="opt.value"
                      type="button"
                      class="type-panel-item"
                      :class="[typeClass(opt.value), { checked: selectedTypeIds.includes(opt.value) }]"
                      @click="toggleTypeFilter(opt.value)"
                    >
                      <span class="type-panel-check">
                        <CheckOutlined v-if="selectedTypeIds.includes(opt.value)" />
                      </span>
                      <i class="type-dot" aria-hidden="true" />
                      <span class="type-panel-label">{{ opt.label }}</span>
                    </button>
                  </div>
                </div>
              </template>
            </Dropdown>
            <span
              v-for="t in visibleTypeTags"
              :key="t.value"
              class="type-tag"
              :class="typeClass(t.value)"
              :title="t.label"
            >
              <i class="type-dot" aria-hidden="true" />
              <span class="type-tag-label">{{ t.label }}</span>
              <CloseOutlined class="type-tag-close" @click="toggleTypeFilter(t.value)" />
            </span>
            <span v-if="hiddenTypeTagCount > 0" class="type-tag type-tag-more">
              +{{ hiddenTypeTagCount }}
            </span>
          </div>
          <div class="filters" role="tablist">
            <button
              type="button"
              class="chip"
              :class="{ active: filter === 'all' }"
              @click="filter = 'all'"
            >
              全部
            </button>
            <button
              type="button"
              class="chip"
              :class="{ active: filter === 'ongoing' }"
              @click="filter = 'ongoing'"
            >
              进行中
            </button>
            <button
              type="button"
              class="chip"
              :class="{ active: filter === 'host' }"
              @click="filter = 'host'"
            >
              我主持
            </button>
            <button
              type="button"
              class="chip"
              :class="{ active: filter === 'joined' }"
              @click="filter = 'joined'"
            >
              我参与
            </button>
          </div>
        </div>

        <div v-if="!loading && !filteredMeetings.length" class="empty-wrap">
          <Empty :description="emptyDescription">
            <Button
              v-if="filter === 'all' && !hasExtraFilter"
              type="primary"
              class="btn-primary"
              @click="openCreate"
            >
              <template #icon><PlusOutlined /></template>
              新建会议室
            </Button>
          </Empty>
        </div>

        <div v-else class="cards">
          <article
            v-for="(m, idx) in filteredMeetings"
            :key="m.id"
            class="meeting-card"
            :class="{ live: isOngoing(m), ended: isEnded(m) }"
            :style="{ '--delay': `${idx * 70}ms` }"
          >
            <div class="card-main">
              <div class="card-top">
                <h3 class="meeting-title">{{ m.title }}</h3>
                <button
                  v-if="canEdit(m)"
                  type="button"
                  class="rename-btn"
                  title="编辑会议"
                  aria-label="编辑会议"
                  @click="openEdit(m)"
                >
                  <EditOutlined />
                </button>
                <span
                  class="pill"
                  :class="isOngoing(m) ? 'pill-live' : isEnded(m) ? 'pill-ended' : 'pill-plan'"
                >
                  {{ statusLabel(m) }}
                </span>
                <span v-if="m.isHost" class="pill pill-host">我主持</span>
                <span v-if="m.typeName" class="pill pill-type" :class="typeClass(m.typeId)">
                  <i class="type-dot" aria-hidden="true" />
                  {{ m.typeName }}
                </span>
              </div>

              <div class="card-meta">
                <span class="meta-item">
                  <span class="meta-label">主持人</span>
                  {{ m.hostName }}
                </span>
                <template v-for="sch in [formatMeetingSchedule(m)]" :key="'sch-' + m.id">
                  <span class="meta-item meta-schedule">
                    <span class="meta-label">会议时间</span>
                    <span class="tabular schedule-text">
                      <span class="schedule-date">{{ sch.date }}</span>
                      <span class="schedule-range">{{ sch.range }}</span>
                    </span>
                  </span>
                  <span v-if="sch.duration && !isOngoing(m)" class="meta-item">
                    <span class="meta-label">会议时长</span>
                    <span class="tabular">{{ sch.duration }}</span>
                  </span>
                </template>
              </div>
              <div v-if="isEnded(m)" class="card-meta card-meta-attendees">
                <span class="meta-item meta-attendees">
                  <span class="meta-label">参会人员</span>
                  {{ attendeesPreview(m) }}
                </span>
              </div>
              <div v-if="isEnded(m) && recordingsOf(m).length" class="card-meta card-meta-recordings">
                <span class="meta-item meta-recordings">
                  <span class="meta-label">录制</span>
                  <span class="recording-list">
                    <span v-for="seg in recordingsOf(m)" :key="seg.id" class="recording-item">
                      第{{ seg.seq }}段
                      <template v-if="canPlaySeg(seg)">
                        <button type="button" class="recording-link" @click="openDetail(m, seg)">回放</button>
                        <button type="button" class="recording-link" @click="downloadSeg(m, seg)">下载</button>
                      </template>
                      <span v-else class="recording-status">{{ recordingStatusText(seg) }}</span>
                    </span>
                  </span>
                </span>
              </div>
              <div v-if="isEnded(m)" class="card-meta card-meta-minutes">
                <span class="meta-item">
                  <span class="meta-label">纪要</span>
                  <button type="button" class="recording-link" @click="openDetail(m)">
                    {{ minutesStatusText(minutesOf(m)?.status) }}
                  </button>
                </span>
              </div>

              <div v-if="isOngoing(m) && progressInfo(m)" class="progress-block">
                <div class="progress-track">
                  <div
                    class="progress-fill"
                    :style="{ width: `${progressInfo(m)!.percent}%` }"
                  />
                </div>
                <div class="progress-text tabular">
                  已进行 {{ progressInfo(m)!.elapsed }} 分钟
                </div>
              </div>
            </div>

            <div v-if="!isEnded(m)" class="card-actions">
              <Button
                v-if="canEnter(m)"
                type="primary"
                class="btn-primary"
                @click="enterMeeting(m)"
              >
                进入
              </Button>
              <Button v-else class="btn-disabled" disabled>未到时间</Button>
              <Button class="btn-invite" @click="openInvite(m)">
                <template #icon><UserAddOutlined /></template>
                邀请
              </Button>
              <Button
                v-if="canEnd(m)"
                class="btn-danger"
                @click="onEnd(m)"
              >
                结束
              </Button>
              <Button
                v-if="canDelete(m)"
                class="btn-danger"
                @click="onDelete(m)"
              >
                删除
              </Button>
            </div>
          </article>
        </div>
      </section>
    </div>

    <Modal
      v-model:open="createOpen"
      title="新建会议室"
      :confirm-loading="creating"
      ok-text="创建"
      cancel-text="取消"
      destroy-on-close
      :afterClose="resetCreateForm"
      @ok="onCreate"
    >
      <Form layout="vertical" class="create-form">
        <Form.Item label="会议名称" required>
          <Input v-model:value="createForm.title" placeholder="例如 周例会" allow-clear />
        </Form.Item>
        <Form.Item label="主持人">
          <Input :value="userLabel" disabled />
        </Form.Item>
        <Form.Item label="会议类型">
          <Select
            v-model:value="createForm.typeId"
            :options="typeFormOptions"
            :disabled="!typeFormOptions.length"
            :placeholder="typeFormOptions.length ? '非必选，不选择则不计类型' : '暂无会议类型'"
            allow-clear
          />
        </Form.Item>
        <Form.Item label="开始时间" required>
          <DatePicker
            v-model:value="createForm.startAt"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="选择开始时间"
            style="width: 100%"
            :disabled-date="(d) => d.isBefore(dayjs().startOf('day'))"
            :disabled-time="disabledCreateStartTime"
          />
        </Form.Item>
        <Form.Item label="结束时间" required>
          <DatePicker
            v-model:value="createForm.endAt"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="选择结束时间"
            style="width: 100%"
            :disabled-date="(d) => !!createForm.startAt && d.isBefore(createForm.startAt, 'day')"
          />
        </Form.Item>
        <Form.Item label="开启录制">
          <div class="record-switch-row">
            <Switch v-model:checked="createForm.recordEnabled" />
            <span class="record-switch-hint">默认关闭；开启后第一位参会者进房自动开始，会中仍可多次启停</span>
          </div>
        </Form.Item>
        <div class="time-hint">
          <InfoCircleOutlined class="time-hint-icon" />
          <ul class="time-hint-list">
            <li>可提前 5 分钟进入；时间仅作说明，可超期进行</li>
            <li>预定结束后主持人可点「结束」；到点无人则自动结束，有人则继续计时</li>
            <li>结束时会自动同步实际结束时间</li>
          </ul>
        </div>
      </Form>
    </Modal>

    <Modal
      v-model:open="editOpen"
      title="编辑会议"
      :confirm-loading="renaming"
      ok-text="保存"
      cancel-text="取消"
      destroy-on-close
      @ok="onEdit"
    >
      <Form layout="vertical" class="create-form">
        <Form.Item label="会议名称" required>
          <Input
            v-model:value="editForm.title"
            placeholder="请输入会议名称"
            :maxlength="64"
            allow-clear
          />
        </Form.Item>
        <Form.Item label="会议类型">
          <Select
            v-model:value="editForm.typeId"
            :options="typeFormOptions"
            :disabled="!typeFormOptions.length"
            :placeholder="typeFormOptions.length ? '非必选，清空则不计类型' : '暂无会议类型'"
            allow-clear
          />
        </Form.Item>
        <Form.Item label="开始时间" required>
          <DatePicker
            v-model:value="editForm.startAt"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="选择开始时间"
            style="width: 100%"
            :disabled-date="(d) => d.isBefore(dayjs().startOf('day'))"
          />
        </Form.Item>
        <Form.Item label="结束时间" required>
          <DatePicker
            v-model:value="editForm.endAt"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="选择结束时间"
            style="width: 100%"
            :disabled-date="(d) => !!editForm.startAt && d.isBefore(editForm.startAt, 'day')"
          />
        </Form.Item>
        <div class="time-hint">
          <InfoCircleOutlined class="time-hint-icon" />
          <ul class="time-hint-list">
            <li>可提前 5 分钟进入；时间仅作说明，可超期进行</li>
            <li>预定结束后主持人可点「结束」；到点无人则自动结束，有人则继续计时</li>
            <li>结束时会自动同步实际结束时间</li>
          </ul>
        </div>
      </Form>
    </Modal>

    <Modal
      v-model:open="inviteOpen"
      :title="inviteTarget?.title || '邀请'"
      :footer="null"
      wrap-class-name="invite-modal"
      destroy-on-close
      :afterClose="clearInviteTarget"
    >
      <div v-if="inviteTarget" class="invite-panel">
        <div class="invite-fields">
          <div class="invite-line">
            <span class="invite-label">主持人</span>
            <span class="invite-value">{{ inviteTarget.hostName }}</span>
          </div>
          <div class="invite-line">
            <span class="invite-label">时间</span>
            <div v-if="inviteTime" class="invite-time">
              <span class="invite-time-date">{{ inviteTime.date }}</span>
              <span class="invite-time-range">
                <ClockCircleOutlined class="invite-time-icon" />
                <span class="tabular">{{ inviteTime.range }}</span>
                <span v-if="inviteTime.duration" class="invite-time-dur">
                  预计时长 {{ inviteTime.duration }}
                </span>
              </span>
            </div>
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
      </div>
    </Modal>

    <Modal
      v-model:open="detailOpen"
      :title="detailMeeting?.title || '会议详情'"
      :footer="null"
      width="760px"
      wrap-class-name="detail-modal"
      destroy-on-close
      :afterClose="closeDetail"
    >
      <div v-if="detailMeeting" class="detail-panel">
        <div class="invite-fields">
          <div class="invite-line">
            <span class="invite-label">主持人</span>
            <span class="invite-value">{{ detailMeeting.hostName }}</span>
          </div>
          <div class="invite-line">
            <span class="invite-label">时间</span>
            <span class="invite-value">{{ formatInviteTimeText(detailMeeting) }}</span>
          </div>
          <div class="invite-line">
            <span class="invite-label">参会人</span>
            <span class="invite-value">{{ attendeesPreview(detailMeeting) }}</span>
          </div>
        </div>

        <div v-if="playSrc" class="detail-player">
          <video
            :key="playSeg?.id"
            class="play-video"
            :src="playSrc"
            controls
            autoplay
            playsinline
            preload="metadata"
            @error="onPlayError"
          />
          <p v-if="playError" class="detail-play-error">{{ playError }}</p>
        </div>
        <p v-else-if="recordingsOf(detailMeeting).some(isProcessingSeg)" class="detail-play-hint">
          录制文件处理中，就绪后将自动可播
        </p>

        <div class="detail-minutes">
          <div class="detail-segs-title">
            会议纪要
            <span class="detail-minutes-status">{{ minutesStatusText(detailMinutes?.status) }}</span>
          </div>
          <p v-if="minutesLoading && !detailMinutes" class="detail-play-hint">纪要加载中…</p>
          <p v-else-if="detailMinutes?.status === 'unavailable'" class="detail-play-hint">
            该会议没有音频，无法生成会议纪要。
          </p>
          <p v-else-if="detailMinutes?.status === 'skipped_empty'" class="detail-play-hint">
            该会议没有音频，无法生成会议纪要。
          </p>
          <p v-else-if="detailMinutes?.status === 'failed'" class="detail-play-hint">
            生成失败：{{ detailMinutes.errorMsg || '未知错误' }}
          </p>
          <p
            v-else-if="detailMinutes && ['pending','transcribing','summarizing'].includes(detailMinutes.status)"
            class="detail-play-hint"
          >
            会议纪要正在编写，完成后会自动更新。
          </p>
          <template v-else-if="detailMinutes?.status === 'ready'">
            <div class="detail-minutes-summary">{{ detailMinutes.summary }}</div>
            <ul v-if="detailMinutes.structured?.todos?.length" class="detail-minutes-list">
              <li v-for="(t, i) in detailMinutes.structured.todos" :key="'todo-'+i">待办：{{ t }}</li>
            </ul>
            <ul v-if="detailMinutes.structured?.decisions?.length" class="detail-minutes-list">
              <li v-for="(t, i) in detailMinutes.structured.decisions" :key="'dec-'+i">决议：{{ t }}</li>
            </ul>
            <ul v-if="detailMinutes.structured?.risks?.length" class="detail-minutes-list">
              <li v-for="(t, i) in detailMinutes.structured.risks" :key="'risk-'+i">风险：{{ t }}</li>
            </ul>
          </template>
          <p v-else class="detail-play-hint">暂无纪要</p>
          <div v-if="detailMeeting.isHost" class="detail-minutes-actions">
            <Button size="small" :loading="minutesRegenning" @click="onRegenerateMinutes">重新生成</Button>
          </div>
        </div>

        <div v-if="recordingsOf(detailMeeting).length" class="detail-segs">
          <div class="detail-segs-title">录制分段</div>
          <div
            v-for="seg in recordingsOf(detailMeeting)"
            :key="seg.id"
            class="detail-seg"
            :class="{ active: playSeg?.id === seg.id }"
          >
            <div class="detail-seg-main">
              <span class="detail-seg-name">第{{ seg.seq }}段</span>
              <span v-if="formatSegSpan(seg)" class="detail-seg-meta">{{ formatSegSpan(seg) }}</span>
              <span v-if="formatFileSize(seg.fileSize)" class="detail-seg-meta">{{ formatFileSize(seg.fileSize) }}</span>
              <span v-if="!canPlaySeg(seg)" class="recording-status">{{ recordingStatusText(seg) }}</span>
            </div>
            <div class="detail-seg-actions">
              <button
                v-if="canPlaySeg(seg)"
                type="button"
                class="recording-link"
                @click="selectPlaySeg(seg)"
              >
                {{ playSeg?.id === seg.id ? '播放中' : '回放' }}
              </button>
              <button
                v-if="canPlaySeg(seg)"
                type="button"
                class="recording-link"
                @click="downloadSeg(detailMeeting, seg)"
              >
                下载
              </button>
            </div>
          </div>
        </div>
      </div>
    </Modal>
    <Teleport to="body">
      <div v-if="minutesHintLoading" class="minutes-settle" role="status" aria-live="polite">
        <div class="minutes-settle-card">
          <div class="minutes-settle-orb" aria-hidden="true">
            <span />
            <span />
            <span />
          </div>
          <p class="minutes-settle-title">会议已结束</p>
          <p class="minutes-settle-sub">{{ minutesHintText }}</p>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.lobby {
  --brand: #f3a04c;
  --brand-strong: #e8892a;
  --live: #10b981;
  --plan: #3b82f6;
  --warn: #f59e0b;
  --danger: #ef4444;
  --ink: rgba(15, 23, 42, 0.92);
  --ink-60: rgba(15, 23, 42, 0.6);
  --ink-35: rgba(15, 23, 42, 0.35);
  --card: #ffffff;
  --line: rgba(15, 23, 42, 0.08);
  --shadow-soft: 0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 24px rgba(15, 23, 42, 0.06);
  --shadow-hover: 0 4px 12px rgba(15, 23, 42, 0.08), 0 16px 32px rgba(15, 23, 42, 0.08);
  --radius: 14px;
  --font: -apple-system, 'PingFang SC', 'Microsoft YaHei', Inter, sans-serif;

  position: relative;
  min-height: 100vh;
  overflow-x: hidden;
  padding: 24px 20px 48px;
  color: var(--ink);
  font-family: var(--font);
  background:
    radial-gradient(900px 480px at 8% -8%, rgba(243, 160, 76, 0.18), transparent 55%),
    radial-gradient(760px 420px at 92% 4%, rgba(79, 70, 229, 0.12), transparent 50%),
    radial-gradient(700px 400px at 50% 100%, rgba(16, 185, 129, 0.1), transparent 55%),
    linear-gradient(180deg, #f7f8fc 0%, #eef2f8 48%, #f5f7fb 100%);
}

html[data-theme='dark'] .lobby {
  --ink: rgba(248, 250, 252, 0.94);
  --ink-60: rgba(226, 232, 240, 0.68);
  --ink-35: rgba(148, 163, 184, 0.55);
  --card: rgba(17, 24, 39, 0.82);
  --line: rgba(148, 163, 184, 0.16);
  --shadow-soft: 0 1px 2px rgba(0, 0, 0, 0.2), 0 10px 28px rgba(0, 0, 0, 0.28);
  --shadow-hover: 0 8px 24px rgba(0, 0, 0, 0.35);
  background:
    radial-gradient(900px 480px at 8% -8%, rgba(243, 160, 76, 0.14), transparent 55%),
    radial-gradient(760px 420px at 92% 4%, rgba(79, 70, 229, 0.14), transparent 50%),
    radial-gradient(700px 400px at 50% 100%, rgba(16, 185, 129, 0.1), transparent 55%),
    linear-gradient(180deg, #0b1220 0%, #111827 50%, #0f172a 100%);
}

.blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  pointer-events: none;
  opacity: 0.55;
}
.blob-a {
  width: 340px;
  height: 340px;
  top: -60px;
  left: -40px;
  background: rgba(243, 160, 76, 0.35);
}
.blob-b {
  width: 300px;
  height: 300px;
  top: 120px;
  right: -80px;
  background: rgba(99, 102, 241, 0.22);
}
.blob-c {
  width: 280px;
  height: 280px;
  bottom: 40px;
  left: 30%;
  background: rgba(16, 185, 129, 0.16);
}

.shell {
  position: relative;
  z-index: 1;
  width: min(1160px, 100%);
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 14px 16px;
  border-radius: var(--radius);
  background: color-mix(in srgb, var(--card) 86%, transparent);
  border: 1px solid var(--line);
  box-shadow: var(--shadow-soft);
  backdrop-filter: blur(14px);
}

.nav-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.logo {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  object-fit: cover;
  flex-shrink: 0;
  box-shadow: 0 8px 18px rgba(243, 160, 76, 0.28);
}

.nav-titles h1 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--ink);
}

.nav-sub {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--ink-60);
}

.user {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.nums em {
  font-style: normal;
  font-weight: 650;
  color: var(--live);
  font-variant-numeric: tabular-nums;
}
.nums em.soft {
  color: var(--ink-60);
}
.slash,
.divider {
  opacity: 0.5;
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.icon-btn {
  width: 32px !important;
  height: 32px !important;
  min-width: 32px !important;
  padding: 0 !important;
  border: none !important;
  color: var(--ink-60) !important;
  background: transparent !important;
  box-shadow: none !important;
}
.icon-btn:hover,
.icon-btn:focus {
  color: var(--ink) !important;
  background: transparent !important;
}

.btn-secondary,
.btn-ghost,
.btn-disabled {
  border-radius: 10px !important;
  border-color: var(--line) !important;
  color: var(--ink-60) !important;
  background: color-mix(in srgb, var(--card) 70%, transparent) !important;
  box-shadow: none !important;
}

.btn-invite {
  border-radius: 10px !important;
  border-color: rgba(59, 130, 246, 0.28) !important;
  background: rgba(59, 130, 246, 0.1) !important;
  color: #2563eb !important;
  box-shadow: none !important;
}
.btn-invite:hover {
  border-color: rgba(59, 130, 246, 0.45) !important;
  background: rgba(59, 130, 246, 0.16) !important;
  color: #1d4ed8 !important;
}
html[data-theme='dark'] .btn-invite {
  border-color: rgba(96, 165, 250, 0.35) !important;
  background: rgba(59, 130, 246, 0.18) !important;
  color: #93c5fd !important;
}
html[data-theme='dark'] .btn-invite:hover {
  background: rgba(59, 130, 246, 0.26) !important;
  color: #bfdbfe !important;
}

.btn-primary {
  border-radius: 10px !important;
  border-color: var(--brand) !important;
  background: var(--brand) !important;
  color: #fff !important;
  box-shadow: 0 8px 18px rgba(243, 160, 76, 0.28) !important;
}
.btn-primary:hover {
  background: var(--brand-strong) !important;
  border-color: var(--brand-strong) !important;
}
.btn-primary:active {
  transform: translateY(1px);
}

.btn-danger {
  border-radius: 10px !important;
  border-color: rgba(239, 68, 68, 0.18) !important;
  background: rgba(239, 68, 68, 0.08) !important;
  color: var(--danger) !important;
}
.btn-danger:hover {
  background: rgba(239, 68, 68, 0.14) !important;
}
html[data-theme='dark'] .btn-danger {
  border-color: rgba(248, 113, 113, 0.45) !important;
  background: rgba(239, 68, 68, 0.18) !important;
  color: #fca5a5 !important;
}
html[data-theme='dark'] .btn-danger:hover {
  border-color: rgba(252, 165, 165, 0.65) !important;
  background: rgba(239, 68, 68, 0.28) !important;
  color: #fecaca !important;
}

.btn-disabled {
  opacity: 0.55 !important;
  cursor: not-allowed !important;
}

/* 顶栏搜索：图标态与输入框态 */
.search-box {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.search-input {
  width: 200px;
  border-radius: 10px !important;
}

.search-input .search-prefix {
  color: var(--ink-35);
}

@media (max-width: 640px) {
  .search-input {
    width: 148px;
  }
}

.stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px;
  border-radius: var(--radius);
  background: var(--card);
  border: 1px solid var(--line);
  box-shadow: var(--shadow-soft);
  animation: rise 0.45s ease both;
  animation-delay: var(--delay, 0ms);
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  font-size: 18px;
}
.stat-icon.green {
  color: var(--live);
  background: rgba(16, 185, 129, 0.12);
}
.stat-icon.blue {
  color: var(--plan);
  background: rgba(59, 130, 246, 0.12);
}
.stat-icon.amber {
  color: var(--warn);
  background: rgba(245, 158, 11, 0.14);
}
.stat-icon.violet {
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.12);
}

.stat-num {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
  color: var(--ink);
}
.stat-label {
  margin-top: 2px;
  font-size: 12px;
  color: var(--ink-35);
}

.list-panel {
  border-radius: 16px;
  background: var(--card);
  border: 1px solid var(--line);
  box-shadow: var(--shadow-soft);
  padding: 16px;
  min-height: 360px;
}

.list-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.list-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  row-gap: 6px;
  min-width: 0;
}
.list-title h2 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.list-title .badge {
  flex-shrink: 0;
}
.badge {
  min-width: 22px;
  height: 22px;
  padding: 0 7px;
  border-radius: 999px;
  display: inline-grid;
  place-items: center;
  font-size: 11px;
  font-weight: 650;
  color: var(--ink-60);
  background: rgba(15, 23, 42, 0.06);
  font-variant-numeric: tabular-nums;
}
html[data-theme='dark'] .badge {
  background: rgba(255, 255, 255, 0.08);
}

/* ========== 会议类型配色（类型ID 取模，标签/下拉项/列表标签一致） ========== */
.type-none {
  --type-fg: #475569;
  --type-bg: rgba(100, 116, 139, 0.12);
  --type-bd: rgba(100, 116, 139, 0.26);
}
.type-c0 {
  /* 靛蓝 */
  --type-fg: #4f46e5;
  --type-bg: rgba(99, 102, 241, 0.12);
  --type-bd: rgba(99, 102, 241, 0.28);
}
.type-c1 {
  /* 琥珀 */
  --type-fg: #b45309;
  --type-bg: rgba(245, 158, 11, 0.15);
  --type-bd: rgba(245, 158, 11, 0.32);
}
.type-c2 {
  /* 翠绿 */
  --type-fg: #047857;
  --type-bg: rgba(16, 185, 129, 0.13);
  --type-bd: rgba(16, 185, 129, 0.3);
}
.type-c3 {
  /* 紫罗兰 */
  --type-fg: #7c3aed;
  --type-bg: rgba(139, 92, 246, 0.12);
  --type-bd: rgba(139, 92, 246, 0.28);
}
.type-c4 {
  /* 蓝 */
  --type-fg: #1d4ed8;
  --type-bg: rgba(59, 130, 246, 0.12);
  --type-bd: rgba(59, 130, 246, 0.28);
}
.type-c5 {
  /* 玫红 */
  --type-fg: #be123c;
  --type-bg: rgba(244, 63, 94, 0.11);
  --type-bd: rgba(244, 63, 94, 0.28);
}
.type-c6 {
  /* 青绿 */
  --type-fg: #0f766e;
  --type-bg: rgba(13, 148, 136, 0.13);
  --type-bd: rgba(13, 148, 136, 0.3);
}
.type-c7 {
  /* 橙 */
  --type-fg: #c2410c;
  --type-bg: rgba(249, 115, 22, 0.13);
  --type-bd: rgba(249, 115, 22, 0.3);
}
.type-c8 {
  /* 品红 */
  --type-fg: #a21caf;
  --type-bg: rgba(217, 70, 239, 0.11);
  --type-bd: rgba(217, 70, 239, 0.28);
}
.type-c9 {
  /* 天青 */
  --type-fg: #0e7490;
  --type-bg: rgba(6, 182, 212, 0.12);
  --type-bd: rgba(6, 182, 212, 0.3);
}

html[data-theme='dark'] .type-none {
  --type-fg: #cbd5e1;
  --type-bg: rgba(148, 163, 184, 0.2);
  --type-bd: rgba(148, 163, 184, 0.36);
}
html[data-theme='dark'] .type-c0 {
  --type-fg: #a5b4fc;
  --type-bg: rgba(99, 102, 241, 0.24);
  --type-bd: rgba(129, 140, 248, 0.4);
}
html[data-theme='dark'] .type-c1 {
  --type-fg: #fcd34d;
  --type-bg: rgba(245, 158, 11, 0.22);
  --type-bd: rgba(252, 211, 77, 0.4);
}
html[data-theme='dark'] .type-c2 {
  --type-fg: #6ee7b7;
  --type-bg: rgba(16, 185, 129, 0.22);
  --type-bd: rgba(52, 211, 153, 0.4);
}
html[data-theme='dark'] .type-c3 {
  --type-fg: #c4b5fd;
  --type-bg: rgba(139, 92, 246, 0.24);
  --type-bd: rgba(167, 139, 250, 0.42);
}
html[data-theme='dark'] .type-c4 {
  --type-fg: #93c5fd;
  --type-bg: rgba(59, 130, 246, 0.24);
  --type-bd: rgba(96, 165, 250, 0.42);
}
html[data-theme='dark'] .type-c5 {
  --type-fg: #fda4af;
  --type-bg: rgba(244, 63, 94, 0.22);
  --type-bd: rgba(251, 113, 133, 0.4);
}
html[data-theme='dark'] .type-c6 {
  --type-fg: #5eead4;
  --type-bg: rgba(13, 148, 136, 0.24);
  --type-bd: rgba(45, 212, 191, 0.4);
}
html[data-theme='dark'] .type-c7 {
  --type-fg: #fdba74;
  --type-bg: rgba(249, 115, 22, 0.22);
  --type-bd: rgba(251, 146, 60, 0.4);
}
html[data-theme='dark'] .type-c8 {
  --type-fg: #f0abfc;
  --type-bg: rgba(217, 70, 239, 0.22);
  --type-bd: rgba(232, 121, 249, 0.4);
}
html[data-theme='dark'] .type-c9 {
  --type-fg: #67e8f9;
  --type-bg: rgba(6, 182, 212, 0.22);
  --type-bd: rgba(34, 211, 238, 0.4);
}

.type-dot {
  display: inline-block;
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--type-fg, currentColor);
}

/* ========== 会议类型筛选：折叠图标按钮 + 下拉面板 ========== */
.type-filter-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--ink-35);
  cursor: pointer;
  transition:
    color 0.18s ease,
    background 0.18s ease;
}

.type-filter-btn:hover {
  color: var(--ink-60);
  background: rgba(15, 23, 42, 0.06);
}

.type-filter-btn:focus-visible {
  outline: none;
  color: var(--ink-60);
  background: rgba(15, 23, 42, 0.06);
  box-shadow: 0 0 0 3px rgba(243, 160, 76, 0.2);
}

.type-filter-btn.open {
  color: var(--ink);
  background: rgba(15, 23, 42, 0.07);
}

.type-filter-btn.active {
  color: var(--brand-strong);
}

.type-filter-btn.active:hover,
.type-filter-btn.active.open {
  background: rgba(243, 160, 76, 0.14);
}

.type-filter-caret {
  transition: transform 0.2s ease;
}

.type-filter-btn.open .type-filter-caret {
  transform: rotate(180deg);
}

/* 有筛选时右上角一个小圆点提示，不做其它装饰 */
.type-filter-dot {
  position: absolute;
  top: 5px;
  right: 5px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--brand);
  box-shadow: 0 0 0 2px var(--card);
}

html[data-theme='dark'] .type-filter-btn:hover,
html[data-theme='dark'] .type-filter-btn:focus-visible {
  background: rgba(255, 255, 255, 0.08);
}

html[data-theme='dark'] .type-filter-btn.open {
  background: rgba(255, 255, 255, 0.1);
}

html[data-theme='dark'] .type-filter-btn.active {
  color: #fcd34d;
}

html[data-theme='dark'] .type-filter-dot {
  box-shadow: 0 0 0 2px #111827;
}


/* 已选类型标签 */
.type-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  padding: 0 7px 0 8px;
  border: 1px solid var(--type-bd);
  border-radius: 999px;
  background: var(--type-bg);
  color: var(--type-fg);
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
}

.type-tag-label {
  max-width: 96px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.type-tag-close {
  font-size: 9px;
  opacity: 0.65;
  cursor: pointer;
  transition: opacity 0.15s ease;
}
.type-tag-close:hover {
  opacity: 1;
}

.type-tag-more {
  padding: 0 9px;
  border-color: var(--line);
  background: rgba(15, 23, 42, 0.05);
  color: var(--ink-60);
}
html[data-theme='dark'] .type-tag-more {
  background: rgba(255, 255, 255, 0.08);
}

/* 下拉面板（挂到 body，需自带主题变量） */
.type-panel {
  --panel-bg: #ffffff;
  --panel-line: rgba(15, 23, 42, 0.08);
  --panel-ink: rgba(15, 23, 42, 0.92);
  --panel-ink-60: rgba(15, 23, 42, 0.6);
  --panel-hover: rgba(15, 23, 42, 0.05);

  min-width: 208px;
  max-width: 280px;
  padding: 6px;
  border: 1px solid var(--panel-line);
  border-radius: 12px;
  background: var(--panel-bg);
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.14);
}

html[data-theme='dark'] .type-panel {
  --panel-bg: #111827;
  --panel-line: rgba(148, 163, 184, 0.18);
  --panel-ink: rgba(248, 250, 252, 0.94);
  --panel-ink-60: rgba(226, 232, 240, 0.68);
  --panel-hover: rgba(255, 255, 255, 0.07);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
}

.type-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 6px 8px 9px;
  margin-bottom: 2px;
  border-bottom: 1px solid var(--panel-line);
}

.type-panel-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--panel-ink-60);
}

.type-panel-clear {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--brand-strong, #e8892a);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}
.type-panel-clear:hover {
  text-decoration: underline;
}

.type-panel-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 320px;
  overflow-y: auto;
}

.type-panel-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 9px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--panel-ink);
  font-size: 13px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s ease;
}

.type-panel-item:hover {
  background: var(--panel-hover);
}

.type-panel-item.checked {
  background: var(--type-bg);
  color: var(--type-fg);
  font-weight: 600;
}

.type-panel-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 15px;
  height: 15px;
  border: 1px solid var(--panel-line);
  border-radius: 5px;
  color: var(--type-fg);
  font-size: 9px;
  background: transparent;
}

.type-panel-item.checked .type-panel-check {
  border-color: var(--type-bd);
  background: var(--type-bg);
}

.type-panel-item .type-dot {
  width: 7px;
  height: 7px;
}

.type-panel-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}



.filters {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.04);
  border: 1px solid var(--line);
}
html[data-theme='dark'] .filters {
  background: rgba(255, 255, 255, 0.04);
}

.chip {
  appearance: none;
  border: 0;
  background: transparent;
  color: var(--ink-60);
  font-size: 12px;
  font-weight: 550;
  padding: 6px 12px;
  border-radius: 9px;
  cursor: pointer;
  transition: all 0.18s ease;
}
.chip:hover {
  color: var(--ink);
}
.chip.active {
  background: var(--card);
  color: var(--ink);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}

.cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meeting-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 16px 18px;
  border-radius: var(--radius);
  background: color-mix(in srgb, var(--card) 92%, #f8fafc);
  border: 1px solid var(--line);
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    border-color 0.2s ease;
  animation: rise 0.45s ease both;
  animation-delay: var(--delay, 0ms);
}
html[data-theme='dark'] .meeting-card {
  background: rgba(255, 255, 255, 0.03);
}

.meeting-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-hover);
  border-color: color-mix(in srgb, var(--brand) 28%, var(--line));
}

.meeting-card.live {
  border-color: rgba(16, 185, 129, 0.22);
  background: color-mix(in srgb, var(--card) 90%, #ecfdf5);
  box-shadow: 0 1px 2px rgba(16, 185, 129, 0.05);
}
html[data-theme='dark'] .meeting-card.live {
  border-color: rgba(52, 211, 153, 0.28);
  background: color-mix(in srgb, rgba(255, 255, 255, 0.03) 72%, rgba(16, 185, 129, 0.12));
  box-shadow: none;
}

.meeting-card.ended {
  opacity: 0.78;
}

.card-main {
  flex: 1;
  min-width: 240px;
}

.card-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.meeting-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--ink);
}

.rename-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition:
    color 0.15s ease,
    background 0.15s ease;
}
.rename-btn:hover {
  color: var(--brand-strong);
  background: color-mix(in srgb, var(--brand) 14%, transparent);
}

.pill {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 650;
  letter-spacing: 0.02em;
}
.pill-live {
  color: #047857;
  background: rgba(16, 185, 129, 0.14);
}
html[data-theme='dark'] .pill-live {
  color: #6ee7b7;
  background: rgba(16, 185, 129, 0.2);
}
.pill-plan {
  color: #1d4ed8;
  background: rgba(59, 130, 246, 0.12);
}
html[data-theme='dark'] .pill-plan {
  color: #93c5fd;
  background: rgba(59, 130, 246, 0.28);
}
.pill-ended {
  color: #64748b;
  background: rgba(148, 163, 184, 0.18);
}
html[data-theme='dark'] .pill-ended {
  color: #cbd5e1;
  background: rgba(148, 163, 184, 0.22);
}
.pill-host {
  color: #b45309;
  background: rgba(245, 158, 11, 0.16);
}
html[data-theme='dark'] .pill-host {
  color: #fcd34d;
  background: rgba(245, 158, 11, 0.22);
}
/* 列表卡片上的类型标签：颜色与筛选标签一致 */
.pill-type {
  gap: 5px;
  padding: 0 9px 0 7px;
  border: 1px solid var(--type-bd);
  color: var(--type-fg);
  background: var(--type-bg);
  max-width: 170px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pill-type .type-dot {
  width: 5px;
  height: 5px;
}

.card-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px;
  font-size: 12px;
  color: var(--ink-60);
}

.card-meta-attendees {
  margin-top: 8px;
}

.card-meta-recordings {
  margin-top: 6px;
}

.meta-attendees,
.meta-recordings {
  min-width: 0;
  align-items: flex-start;
  line-height: 1.45;
}

.recording-list {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 10px 14px;
}

.recording-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.recording-link {
  color: var(--brand-strong, #e8892a);
  cursor: pointer;
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  text-decoration: none;
}

.recording-link:hover {
  text-decoration: underline;
}

.recording-status {
  color: var(--vc-muted);
}

.play-video {
  display: block;
  width: 100%;
  max-height: 70vh;
  background: #000;
  border-radius: 10px;
}

.detail-panel {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.detail-player {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-play-error {
  margin: 0;
  font-size: 12px;
  color: var(--vc-danger, #ef4444);
}

.detail-play-hint {
  margin: 0;
  font-size: 13px;
  color: var(--vc-muted);
}

.detail-segs {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-segs-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--vc-ink);
}

.detail-seg {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--vc-line);
  background: color-mix(in srgb, var(--vc-panel-solid) 88%, transparent);
}

.detail-seg.active {
  border-color: rgba(243, 160, 76, 0.45);
  background: rgba(243, 160, 76, 0.1);
}

.detail-seg-main {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  min-width: 0;
}

.detail-seg-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--vc-ink);
}

.detail-seg-meta {
  font-size: 12px;
  color: var(--vc-muted);
}

.detail-seg-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.meta-item.muted {
  color: var(--ink-35);
}

.schedule-text {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 8px;
}

.schedule-date,
.schedule-range {
  white-space: nowrap;
}

.meta-label {
  position: relative;
  padding-right: 8px;
  margin-right: 2px;
  color: var(--brand-strong);
  font-size: 12px;
  font-weight: 600;
}

.meta-label::after {
  content: '';
  position: absolute;
  top: 50%;
  right: 0;
  width: 1px;
  height: 10px;
  transform: translateY(-50%);
  background: var(--line);
}

.tabular {
  font-variant-numeric: tabular-nums;
}

.progress-block {
  margin-top: 12px;
  max-width: 420px;
}

.progress-track {
  height: 4px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.08);
  overflow: hidden;
}
html[data-theme='dark'] .progress-track {
  background: rgba(255, 255, 255, 0.1);
}
.progress-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #34d399, #10b981);
}
.progress-text {
  margin-top: 6px;
  font-size: 11px;
  color: var(--ink-35);
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.empty-wrap {
  padding: 56px 12px;
}

.create-form {
  margin-top: 8px;
}

.record-switch-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.record-switch-hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--vc-muted, #64748b);
  flex: 1;
}

.time-hint {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  margin: 4px 0 0;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid rgba(243, 160, 76, 0.28);
  background: rgba(243, 160, 76, 0.1);
  color: rgba(15, 23, 42, 0.62);
  font-size: 12px;
  line-height: 1.55;
}
html[data-theme='dark'] .time-hint {
  border-color: rgba(243, 160, 76, 0.32);
  background: rgba(243, 160, 76, 0.12);
  color: rgba(226, 232, 240, 0.72);
}
.time-hint-icon {
  margin-top: 2px;
  flex-shrink: 0;
  color: #e8892a;
  font-size: 14px;
}
.time-hint-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.time-hint-list li {
  position: relative;
  padding-left: 12px;
}
.time-hint-list li::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0.55em;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: #f3a04c;
}

.invite-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
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
  color: #f3a04c;
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

@media (max-width: 520px) {
  .invite-actions {
    flex-direction: column;
  }
}

@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 900px) {
  .stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .lobby {
    padding: 14px 12px 32px;
  }
  .stats {
    grid-template-columns: 1fr 1fr;
  }
  .meeting-card {
    padding: 14px;
  }
  .card-actions {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>

<style>
/* 会议类型筛选面板：去掉 ant Dropdown 默认外观，交给 .type-panel 自己画 */
.type-panel-dropdown {
  padding: 0 !important;
  background: transparent !important;
  box-shadow: none !important;
}

.invite-modal .ant-modal-content,
.detail-modal .ant-modal-content {
  overflow: hidden;
  border-radius: 14px;
}

.minutes-settle {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(15, 23, 42, 0.28);
  backdrop-filter: blur(10px);
}
html[data-theme='dark'] .minutes-settle {
  background: rgba(2, 6, 23, 0.55);
}

.minutes-settle-card {
  width: min(360px, 100%);
  padding: 28px 24px 24px;
  border-radius: 18px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: #fff;
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.18);
  text-align: center;
}
html[data-theme='dark'] .minutes-settle-card {
  border-color: rgba(148, 163, 184, 0.18);
  background: #111827;
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.45);
}

.minutes-settle-orb {
  position: relative;
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
}
.minutes-settle-orb span {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid transparent;
  border-top-color: #f3a04c;
  border-right-color: rgba(243, 160, 76, 0.35);
  animation: minutes-settle-spin 1s linear infinite;
}
.minutes-settle-orb span:nth-child(2) {
  inset: 8px;
  animation-duration: 1.4s;
  animation-direction: reverse;
  border-top-color: #10b981;
  border-right-color: rgba(16, 185, 129, 0.3);
}
.minutes-settle-orb span:nth-child(3) {
  inset: 16px;
  animation-duration: 0.8s;
  border-top-color: #f3a04c;
  opacity: 0.7;
}

.minutes-settle-title {
  margin: 0;
  font-size: 16px;
  font-weight: 650;
  color: rgba(15, 23, 42, 0.92);
}
html[data-theme='dark'] .minutes-settle-title {
  color: rgba(248, 250, 252, 0.94);
}

.minutes-settle-sub {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.55;
  color: rgba(15, 23, 42, 0.6);
}
html[data-theme='dark'] .minutes-settle-sub {
  color: rgba(226, 232, 240, 0.68);
}

@keyframes minutes-settle-spin {
  to {
    transform: rotate(360deg);
  }
}

.invite-modal .ant-modal-header,
.detail-modal .ant-modal-header {
  margin: 0;
  padding: 20px 24px 8px;
  border-bottom: none;
}

.invite-modal .ant-modal-title,
.detail-modal .ant-modal-title {
  color: var(--vc-ink);
  font-size: 17px;
  font-weight: 650;
  line-height: 1.35;
}

.invite-modal .ant-modal-close,
.detail-modal .ant-modal-close {
  top: 16px;
}

.invite-modal .ant-modal-body,
.detail-modal .ant-modal-body {
  padding: 4px 24px 22px;
}

.detail-minutes {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid rgba(127, 127, 127, 0.2);
}
.detail-minutes-status {
  margin-left: 8px;
  font-weight: 400;
  opacity: 0.75;
  font-size: 13px;
}
.detail-minutes-summary {
  white-space: pre-wrap;
  line-height: 1.6;
  margin: 8px 0 12px;
}
.detail-minutes-list {
  margin: 0 0 8px;
  padding-left: 1.2em;
}
.detail-minutes-actions {
  margin-top: 8px;
}
.card-meta-minutes {
  margin-top: 10px;
}
.card-meta-minutes .recording-link {
  margin-left: 4px;
}
</style>
