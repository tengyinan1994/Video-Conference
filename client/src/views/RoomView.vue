<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Alert, Button, Space, Tag, message } from 'ant-design-vue'
import {
  AudioMutedOutlined,
  AudioOutlined,
  DesktopOutlined,
  VideoCameraOutlined,
  VideoCameraAddOutlined,
  LogoutOutlined,
} from '@ant-design/icons-vue'
import MediaTrack from '@/components/MediaTrack.vue'
import { useLiveKitRoom } from '@/composables/useLiveKitRoom'

interface SessionPayload {
  serverUrl: string
  token: string
  room: string
  identity: string
  nickname: string
  expiresAt: number
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
  connect,
  disconnect,
  toggleMic,
  toggleCamera,
  toggleScreenShare,
} = useLiveKitRoom()

const session = ref<SessionPayload | null>(null)
const joining = ref(false)

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
    await connect(parsed.serverUrl, parsed.token)
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

onMounted(() => {
  void enter()
})
</script>

<template>
  <div class="room">
    <header class="top">
      <div>
        <strong>房间 {{ route.params.room }}</strong>
        <Tag class="tag" :color="status === 'connected' ? 'success' : 'processing'">
          {{ statusText }}
        </Tag>
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

    <div class="grid">
      <div v-for="p in participants" :key="p.identity" class="tile">
        <MediaTrack
          v-if="p.videoTrack"
          :track="p.videoTrack"
          :mirror="p.isLocal && !screenSharing"
          muted
        />
        <div v-else class="placeholder">{{ p.name.slice(0, 1) }}</div>
        <MediaTrack
          v-if="p.audioTrack && !p.isLocal"
          :track="p.audioTrack"
        />
        <div class="label">
          {{ p.name }}
          <span v-if="p.isLocal">（我）</span>
          <span v-if="!p.isMicrophoneEnabled"> · 静音</span>
        </div>
      </div>
      <div v-if="!participants.length" class="empty">
        {{ joining || status === 'connecting' ? '正在进入房间…' : '暂无参与者' }}
      </div>
    </div>

    <footer class="controls">
      <Space>
        <Button @click="toggleMic">
          <template #icon>
            <AudioMutedOutlined v-if="!micEnabled" />
            <AudioOutlined v-else />
          </template>
          {{ micEnabled ? '静音' : '取消静音' }}
        </Button>
        <Button @click="toggleCamera">
          <template #icon>
            <VideoCameraOutlined v-if="cameraEnabled" />
            <VideoCameraAddOutlined v-else />
          </template>
          {{ cameraEnabled ? '关摄像头' : '开摄像头' }}
        </Button>
        <Button @click="toggleScreenShare">
          <template #icon>
            <DesktopOutlined />
          </template>
          {{ screenSharing ? '停止共享' : '屏幕共享' }}
        </Button>
        <Button danger type="primary" @click="leave">
          <template #icon>
            <LogoutOutlined />
          </template>
          离开
        </Button>
      </Space>
    </footer>
  </div>
</template>

<style scoped>
.room {
  min-height: 100vh;
  background: #0b1220;
  color: #e2e8f0;
  display: flex;
  flex-direction: column;
}
.top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #1e293b;
}
.tag {
  margin-left: 8px;
}
.meta {
  color: #94a3b8;
  font-size: 13px;
}
.banner {
  margin: 12px 20px 0;
}
.grid {
  flex: 1;
  display: grid;
  gap: 12px;
  padding: 16px 20px 96px;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  align-content: start;
}
.tile {
  position: relative;
  aspect-ratio: 16 / 10;
  background: #111827;
  border-radius: 12px;
  overflow: hidden;
}
.placeholder {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
  font-size: 48px;
  color: #64748b;
  background: #111827;
}
.label {
  position: absolute;
  left: 10px;
  bottom: 10px;
  padding: 4px 8px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.75);
  font-size: 12px;
}
.empty {
  color: #94a3b8;
  padding: 40px 0;
}
.controls {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: center;
  padding: 16px;
  background: rgba(15, 23, 42, 0.92);
  border-top: 1px solid #1e293b;
}
</style>
