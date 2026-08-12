<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Button,
  DatePicker,
  Empty,
  Form,
  Input,
  Modal,
  message,
} from 'ant-design-vue'
import {
  CalendarOutlined,
  ClockCircleOutlined,
  CopyOutlined,
  EditOutlined,
  HistoryOutlined,
  LogoutOutlined,
  PlusOutlined,
  ReloadOutlined,
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
  updateMeeting,
  type MeetingItem,
} from '@/api/conference'
import { clearAuth, displayName, getAuth, setAuth, subscribeAuth } from '@/stores/auth'
import { ApiError } from '@/utils/request'

type FilterKey = 'all' | 'ongoing' | 'host'

const router = useRouter()
const loading = ref(false)
const creating = ref(false)
const createOpen = ref(false)
const renaming = ref(false)
const editOpen = ref(false)
const inviteOpen = ref(false)
const inviteTarget = ref<MeetingItem | null>(null)
const meetings = ref<MeetingItem[]>([])
const userLabel = ref(displayName())
const filter = ref<FilterKey>('all')

type InviteKind = 'guest' | 'member'

const inviteTime = computed(() =>
  inviteTarget.value ? formatInviteTime(inviteTarget.value) : null,
)

const createForm = reactive({
  title: '',
  startAt: undefined as Dayjs | undefined,
  endAt: undefined as Dayjs | undefined,
})

const editForm = reactive({
  id: 0,
  title: '',
  startAt: undefined as Dayjs | undefined,
  endAt: undefined as Dayjs | undefined,
})

const sortedMeetings = computed(() => {
  const order: Record<string, number> = { ongoing: 0, scheduled: 1, ended: 2 }
  const list = [...meetings.value]
  list.sort((a, b) => {
    const ao = order[a.tab] ?? 9
    const bo = order[b.tab] ?? 9
    if (ao !== bo) return ao - bo
    return dayjs(a.startAt).valueOf() - dayjs(b.startAt).valueOf()
  })
  return list
})

const filteredMeetings = computed(() => {
  const list = sortedMeetings.value
  if (filter.value === 'ongoing') return list.filter((m) => m.tab === 'ongoing')
  if (filter.value === 'host') return list.filter((m) => m.isHost)
  return list
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

async function refresh() {
  loading.value = true
  try {
    const data = await listMeetings('all')
    meetings.value = data.list ?? []
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '加载会议列表失败')
  } finally {
    loading.value = false
  }
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
  return (m.attendees ?? []).filter((n) => !!n?.trim())
}

function attendeesPreview(m: MeetingItem) {
  const list = attendeesOf(m)
  if (!list.length) return '暂无'
  if (list.length <= 6) return list.join('、')
  return `${list.slice(0, 6).join('、')} 等 ${list.length} 人`
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

/** 主持人：已可进入（含提前 5 分钟）即可结束，不必等状态变成「进行中」 */
function canEnd(m: MeetingItem) {
  return m.isHost && !isEnded(m) && canEnter(m)
}

/** 主持人：仅未开始（未到可进窗口）的预定会议可删；已结束只能在管理端删 */
function canDelete(m: MeetingItem) {
  return m.isHost && !isEnded(m) && !canEnter(m)
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
    sessionStorage.setItem(
      'vc.session',
      JSON.stringify({
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
      }),
    )
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
        await endMeeting(m.id)
        message.success('会议已结束')
        await refresh()
      } catch (err) {
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
  renaming.value = true
  try {
    await updateMeeting({
      id: editForm.id,
      title,
      startAt: editForm.startAt.format('YYYY-MM-DD HH:mm:ss'),
      endAt: editForm.endAt.format('YYYY-MM-DD HH:mm:ss'),
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
  creating.value = true
  try {
    await createMeeting({
      title: createForm.title.trim(),
      hostName: displayName(),
      startAt: createForm.startAt.format('YYYY-MM-DD HH:mm:ss'),
      endAt: createForm.endAt.format('YYYY-MM-DD HH:mm:ss'),
    })
    message.success('会议室已创建')
    createOpen.value = false
    createForm.title = ''
    const now = dayjs()
    createForm.startAt = now.add(5, 'minute')
    createForm.endAt = now.add(1, 'hour')
    await refresh()
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '创建失败')
  } finally {
    creating.value = false
  }
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
  const range = end ? `${start.format('HH:mm')} – ${end.format('HH:mm')}` : start.format('HH:mm')
  const duration = end ? formatDurationMinutes(end.diff(start, 'minute')) : ''
  return { date, range, duration }
}

function formatInviteTimeText(m: MeetingItem) {
  const t = formatInviteTime(m)
  if (!t.duration) return `${t.date} ${t.range}`
  return `${t.date} ${t.range}，预计时长 ${t.duration}`
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
  const start = dayjs(m.startAt)
  const end = dayjs(m.endAt)
  const now = dayjs()
  const total = Math.max(end.diff(start, 'minute'), 1)
  const elapsed = Math.min(Math.max(now.diff(start, 'minute'), 0), total)
  const remain = Math.max(end.diff(now, 'minute'), 0)
  const percent = Math.min(100, Math.round((elapsed / total) * 100))
  return { elapsed, remain, percent }
}

onMounted(() => {
  unsubAuth = subscribeAuth(syncUser)
  void ensureMe()
  void refresh()
  const now = dayjs()
  createForm.startAt = now.add(5, 'minute')
  createForm.endAt = now.add(1, 'hour')
})

onUnmounted(() => {
  unsubAuth?.()
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
          <ThemeToggle />
          <Button type="text" class="icon-btn" :loading="loading" @click="refresh">
            <template #icon><ReloadOutlined /></template>
          </Button>
          <Button type="primary" class="btn-primary" @click="createOpen = true">
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
            <h2>全部会议</h2>
            <span class="badge">{{ filteredMeetings.length }}</span>
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
          </div>
        </div>

        <div v-if="!loading && !filteredMeetings.length" class="empty-wrap">
          <Empty description="还没有会议，创建一个开始协作吧">
            <Button type="primary" class="btn-primary" @click="createOpen = true">
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
                  <span v-if="sch.duration" class="meta-item">
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

              <div v-if="isOngoing(m) && progressInfo(m)" class="progress-block">
                <div class="progress-track">
                  <div
                    class="progress-fill"
                    :style="{ width: `${progressInfo(m)!.percent}%` }"
                  />
                </div>
                <div class="progress-text tabular">
                  已进行 {{ progressInfo(m)!.elapsed }} 分钟 · 剩余 {{ progressInfo(m)!.remain }} 分钟
                </div>
              </div>
            </div>

            <div class="card-actions">
              <Button
                v-if="canEnter(m)"
                type="primary"
                class="btn-primary"
                @click="enterMeeting(m)"
              >
                进入
              </Button>
              <Button v-else-if="isEnded(m)" class="btn-disabled" disabled>已结束</Button>
              <Button v-else class="btn-disabled" disabled>未到时间</Button>
              <Button v-if="!isEnded(m)" class="btn-invite" @click="openInvite(m)">
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
      @ok="onCreate"
    >
      <Form layout="vertical" class="create-form">
        <Form.Item label="会议名称" required>
          <Input v-model:value="createForm.title" placeholder="例如 周例会" allow-clear />
        </Form.Item>
        <Form.Item label="主持人">
          <Input :value="userLabel" disabled />
        </Form.Item>
        <Form.Item label="开始时间" required>
          <DatePicker
            v-model:value="createForm.startAt"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="选择开始时间"
            style="width: 100%"
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
        <p class="time-hint">
          可提前 5 分钟进入会议室；会议可超期进行，时间仅作说明。点击「结束」会自动同步实际结束时间。
        </p>
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
        <Form.Item label="开始时间" required>
          <DatePicker
            v-model:value="editForm.startAt"
            show-time
            format="YYYY-MM-DD HH:mm"
            placeholder="选择开始时间"
            style="width: 100%"
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
        <p class="time-hint">
          可提前 5 分钟进入会议室；会议可超期进行，时间仅作说明。点击「结束」会自动同步实际结束时间。
        </p>
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

.btn-disabled {
  opacity: 0.55 !important;
  cursor: not-allowed !important;
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
}
.list-title h2 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
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
.pill-ended {
  color: #64748b;
  background: rgba(148, 163, 184, 0.18);
}
.pill-host {
  color: #b45309;
  background: rgba(245, 158, 11, 0.16);
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

.meta-attendees {
  min-width: 0;
  align-items: flex-start;
  line-height: 1.45;
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

.time-hint {
  margin: -4px 0 0;
  padding: 0 1px;
  color: rgba(15, 23, 42, 0.4);
  font-size: 12px;
  line-height: 1.55;
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
  color: rgba(15, 23, 42, 0.4);
  font-size: 13px;
  line-height: 1.5;
}

.invite-value {
  color: rgba(15, 23, 42, 0.9);
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
  color: rgba(15, 23, 42, 0.9);
  font-size: 14px;
  font-weight: 560;
  line-height: 1.45;
}

.invite-time-range {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  color: rgba(15, 23, 42, 0.62);
  font-size: 13px;
  line-height: 1.45;
}

.invite-time-icon {
  font-size: 13px;
  color: #f3a04c;
}

.invite-time-dur {
  color: rgba(15, 23, 42, 0.45);
}

.invite-time-dur::before {
  content: '';
  display: inline-block;
  width: 1px;
  height: 11px;
  margin-right: 10px;
  vertical-align: -1px;
  background: rgba(15, 23, 42, 0.12);
}

.invite-actions {
  display: flex;
  flex-direction: row;
  align-items: stretch;
  gap: 12px;
  padding-top: 4px;
  border-top: 1px solid rgba(15, 23, 42, 0.06);
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
  color: rgba(15, 23, 42, 0.35);
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
.invite-modal .ant-modal-content {
  overflow: hidden;
  border-radius: 14px;
}

.invite-modal .ant-modal-header {
  margin: 0;
  padding: 20px 24px 8px;
  border-bottom: none;
}

.invite-modal .ant-modal-title {
  color: rgba(15, 23, 42, 0.92);
  font-size: 17px;
  font-weight: 650;
  line-height: 1.35;
}

.invite-modal .ant-modal-close {
  top: 16px;
}

.invite-modal .ant-modal-body {
  padding: 4px 24px 22px;
}
</style>
