<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Alert,
  Button,
  Drawer,
  Input,
  Select,
  Space,
  Tag,
  Tooltip,
  message,
} from 'ant-design-vue'
import {
  AudioMutedOutlined,
  AudioOutlined,
  DesktopOutlined,
  LogoutOutlined,
  MessageOutlined,
  SoundOutlined,
  TeamOutlined,
  UserDeleteOutlined,
  VideoCameraAddOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons-vue'
import MediaTrack from '@/components/MediaTrack.vue'
import QualityBars from '@/components/QualityBars.vue'
import { kickParticipant, muteAllParticipants, claimHost } from '@/api/conference'
import { ApiError } from '@/utils/request'
import { useLiveKitRoom } from '@/composables/useLiveKitRoom'

interface SessionPayload {
  serverUrl: string
  token: string
  room: string
  identity: string
  nickname: string
  expiresAt: number
  isHost: boolean
  enableMic?: boolean
  enableCamera?: boolean
}

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
const hostActing = ref(false)

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
    case 'error':
      return '连接失败'
    default:
      return '未连接'
  }
})

const isHost = computed(() => !!session.value?.isHost)

async function syncHostRole() {
  if (!session.value || status.value !== 'connected') return
  if (session.value.isHost) return
  try {
    const claimed = await claimHost(session.value.room, session.value.identity)
    if (claimed.isHost && session.value) {
      session.value = { ...session.value, isHost: true }
      sessionStorage.setItem('vc.session', JSON.stringify(session.value))
    }
  } catch {
    // ignore
  }
}

const avatarParticipants = computed(() => participants.value)

const sideParticipants = computed(() => participants.value)

const mainStageTrack = computed(() => {
  const p = speakerParticipant.value
  if (!p) return undefined
  // 投屏时主画面必须是屏幕轨，不要被人像抢走
  if (p.isScreenSharing) return p.screenTrack ?? p.videoTrack
  return p.videoTrack ?? p.cameraTrack
})

const mainIsScreen = computed(() => !!speakerParticipant.value?.isScreenSharing)

const micOptions = computed(() =>
  audioInputs.value.map((d) => ({ value: d.deviceId, label: d.label })),
)
const cameraOptions = computed(() =>
  videoInputs.value.map((d) => ({ value: d.deviceId, label: d.label })),
)
const speakerOptions = computed(() =>
  audioOutputs.value.map((d) => ({ value: d.deviceId, label: d.label })),
)

async function enter() {
  const raw = sessionStorage.getItem('vc.session')
  if (!raw) {
    message.warning('缺少进房凭证，请重新加入')
    await router.replace({ name: 'join' })
    return
  }
  const parsed = JSON.parse(raw) as SessionPayload
  if (parsed.room !== route.params.room) {
    message.warning('房间不匹配，请重新加入')
    await router.replace({ name: 'join' })
    return
  }
  if (parsed.expiresAt * 1000 < Date.now()) {
    message.error('票据已过期，请重新进入')
    sessionStorage.removeItem('vc.session')
    await router.replace({ name: 'join' })
    return
  }
  session.value = parsed
  joining.value = true
  try {
    await connect(parsed.serverUrl, parsed.token, {
      enableMic: !!parsed.enableMic,
      enableCamera: !!parsed.enableCamera,
    })
    await refreshDevices()
    await syncHostRole()
  } catch {
    // errorMessage already set
  } finally {
    joining.value = false
  }
}

async function leave() {
  await disconnect()
  sessionStorage.removeItem('vc.session')
  await router.replace({ name: 'join' })
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
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '全员静音失败')
  } finally {
    hostActing.value = false
  }
}

async function onToggleMic() {
  try {
    await toggleMic()
  } catch (err) {
    message.error(err instanceof Error ? err.message : '无法开关麦克风')
  }
}

async function onToggleCamera() {
  try {
    await toggleCamera()
  } catch (err) {
    message.error(err instanceof Error ? err.message : '无法开关摄像头')
  }
}

async function onToggleScreenShare() {
  try {
    await toggleScreenShare()
  } catch (err) {
    message.error(err instanceof Error ? err.message : '无法开关屏幕共享')
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

// 其他人离开后只剩自己 → 自动接任主持
watch(
  () => participants.value.length,
  (n) => {
    if (n === 1 && status.value === 'connected' && session.value && !session.value.isHost) {
      void syncHostRole()
    }
  },
)

onMounted(() => {
  void enter()
})
</script>

<template>
  <div class="room">
    <header class="top">
      <div class="top-left">
        <strong>房间 {{ route.params.room }}</strong>
        <Tag class="tag" :color="status === 'connected' ? 'success' : 'processing'">
          {{ statusText }}
        </Tag>
        <Tag v-if="isHost" color="gold">主持人</Tag>
      </div>
      <div v-if="session" class="meta">
        {{ session.nickname }} · {{ session.identity }}
      </div>
    </header>

    <Alert
      v-if="errorMessage"
      type="error"
      show-icon
      :message="errorMessage"
      class="banner"
    />
    <Alert
      v-else-if="status === 'reconnecting'"
      type="warning"
      show-icon
      message="网络波动，正在重连…"
      class="banner"
    />

    <div class="stage" :class="layoutMode">
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
            <MediaTrack v-if="p.audioTrack && !p.isLocal" :track="p.audioTrack" />
          </div>
          <div v-if="!avatarParticipants.length" class="empty">
            {{ joining || status === 'connecting' ? '正在进入房间…' : '暂无参与者' }}
          </div>
        </div>
      </template>

      <!-- 有人开摄像头或投屏：演讲者布局，点击右侧可切换主视图 -->
      <template v-else>
        <div class="speaker-main">
          <div
            v-if="speakerParticipant"
            class="tile main"
            :class="{ speaking: speakerParticipant.isSpeaking && !mainIsScreen }"
          >
            <MediaTrack
              v-if="mainStageTrack"
              :track="mainStageTrack"
              :mirror="speakerParticipant.isLocal && !mainIsScreen"
              :fit="mainIsScreen ? 'contain' : 'cover'"
              muted
            />
            <div v-else class="placeholder">
              <span class="placeholder-name">{{ speakerParticipant.name }}</span>
            </div>
            <MediaTrack
              v-if="speakerParticipant.audioTrack && !speakerParticipant.isLocal"
              :track="speakerParticipant.audioTrack"
            />
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
        <div class="speaker-side">
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
              v-if="p.screenTrack || p.cameraTrack"
              :track="p.screenTrack || p.cameraTrack"
              :mirror="p.isLocal && !p.screenTrack"
              :fit="p.screenTrack ? 'contain' : 'cover'"
              muted
            />
            <div v-else class="placeholder sm">
              <span class="placeholder-name">{{ p.name }}</span>
            </div>
            <MediaTrack v-if="p.audioTrack && !p.isLocal" :track="p.audioTrack" />
            <div class="label">
              {{ p.name }}
              <span v-if="p.isScreenSharing"> · 共享</span>
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
        <Button @click="chatOpen = true">
          <template #icon>
            <MessageOutlined />
          </template>
          聊天
        </Button>
        <Button
          v-if="isHost"
          :loading="hostActing"
          @click="onMuteAll"
        >
          <template #icon>
            <SoundOutlined />
          </template>
          全员静音
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
  </div>
</template>

<style scoped>
.room {
  height: 100dvh;
  max-height: 100dvh;
  overflow: hidden;
  background: #0b1220;
  color: #e2e8f0;
  display: flex;
  flex-direction: column;
}
.top {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid #1e293b;
}
.top-left {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.tag {
  margin-left: 8px;
}
.meta {
  color: #94a3b8;
  font-size: 13px;
}
.banner {
  flex-shrink: 0;
  margin: 12px 20px 0;
}
.stage {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 16px 20px 8px;
  display: flex;
  flex-direction: column;
}
.stage.speaker {
  display: grid;
  grid-template-columns: 1fr 180px;
  gap: 12px;
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
  background: #111827;
  border: 2px solid transparent;
  transition: border-color 0.15s ease;
}
.avatar-card.speaking {
  border-color: #22c55e;
}
.avatar-card.local {
  background: #0f172a;
}
.avatar-circle {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: 28px;
  font-weight: 600;
  color: #e2e8f0;
  background: linear-gradient(145deg, #334155, #1e293b);
}
.avatar-name {
  font-size: 14px;
  font-weight: 500;
  text-align: center;
  word-break: break-word;
  max-width: 100%;
}
.avatar-meta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 11px;
  color: #94a3b8;
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
  color: #64748b;
  text-align: center;
}
.tile.side.clickable {
  cursor: pointer;
}
.tile.side.clickable:hover {
  border-color: #38bdf8;
}
.tile.side.active {
  border-color: #38bdf8;
  box-shadow: 0 0 0 1px rgba(56, 189, 248, 0.35);
}
.tile {
  position: relative;
  min-height: 0;
  height: 100%;
  background: #111827;
  border-radius: 12px;
  overflow: hidden;
  border: 2px solid transparent;
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
  border-color: #22c55e;
  box-shadow: 0 0 0 1px rgba(34, 197, 94, 0.35);
}
.placeholder {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
  padding: 16px;
  color: #94a3b8;
  background: #111827;
}
.placeholder-name {
  max-width: 100%;
  font-size: clamp(18px, 3.2vw, 36px);
  font-weight: 600;
  line-height: 1.3;
  text-align: center;
  word-break: break-word;
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
  background: rgba(15, 23, 42, 0.75);
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  max-width: calc(100% - 20px);
}
.empty {
  color: #94a3b8;
  padding: 40px 0;
}
.controls {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 12px 16px 16px;
  background: rgba(15, 23, 42, 0.92);
  border-top: 1px solid #1e293b;
}
.devices {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
  justify-content: center;
}
.device-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.device-label {
  font-size: 12px;
  color: #94a3b8;
  line-height: 1;
}
.member-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 10px 0;
  border-bottom: 1px solid #f1f5f9;
}
.member-name {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  font-weight: 500;
}
.member-sub {
  color: #64748b;
  font-size: 12px;
  margin-top: 2px;
}
.chat-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  height: 100%;
}
.chat-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
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
  color: #64748b;
}
.chat-time {
  font-size: 11px;
  color: #94a3b8;
}
.chat-bubble {
  padding: 8px 12px;
  border-radius: 12px 12px 12px 4px;
  background: #f1f5f9;
  color: #0f172a;
  word-break: break-word;
  line-height: 1.45;
  font-size: 14px;
}
.chat-row.mine .chat-bubble {
  border-radius: 12px 12px 4px 12px;
  background: #1677ff;
  color: #fff;
}
.chat-empty {
  color: #94a3b8;
  text-align: center;
  padding: 48px 0;
  margin: auto;
}
.chat-input {
  flex-shrink: 0;
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid #e2e8f0;
  background: #fff;
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
}
</style>
