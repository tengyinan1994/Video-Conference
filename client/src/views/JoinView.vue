<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button, Card, Form, Input, Switch, Alert, message } from 'ant-design-vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { createToken, shareView, type MeetingShareView } from '@/api/conference'
import { displayName, isLoggedIn } from '@/stores/auth'
import { ApiError } from '@/utils/request'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const loadingInfo = ref(true)
const info = ref<MeetingShareView | null>(null)
const infoError = ref('')

const shareCode = computed(() => String(route.params.shareCode || ''))
const loggedIn = computed(() => isLoggedIn())

const form = reactive({
  nickname: '',
  enableMic: false,
  enableCamera: false,
})

async function loadInfo() {
  loadingInfo.value = true
  infoError.value = ''
  try {
    info.value = await shareView(shareCode.value)
    if (loggedIn.value && !form.nickname) {
      form.nickname = displayName()
    }
  } catch (err) {
    info.value = null
    infoError.value = err instanceof ApiError ? err.message : '无法加载会议信息'
  } finally {
    loadingInfo.value = false
  }
}

async function onJoin() {
  if (!form.nickname.trim()) {
    message.warning('请填写昵称')
    return
  }
  if (!info.value?.canJoin) {
    message.error('当前会议不可加入')
    return
  }
  loading.value = true
  try {
    const data = await createToken({
      shareCode: shareCode.value,
      nickname: form.nickname.trim(),
    })
    sessionStorage.setItem(
      'vc.session',
      JSON.stringify({
        serverUrl: data.serverUrl,
        token: data.token,
        room: data.room,
        title: data.title || info.value?.title,
        identity: data.identity,
        nickname: data.nickname,
        expiresAt: data.expiresAt,
        isHost: !!data.isHost,
        enableMic: form.enableMic,
        enableCamera: form.enableCamera,
        fromShare: true,
        shareCode: shareCode.value,
      }),
    )
    await router.push({ name: 'room', params: { room: data.room } })
  } catch (err) {
    const msg = err instanceof ApiError ? err.message : '获取进房凭证失败'
    message.error(msg)
  } finally {
    loading.value = false
  }
}

function goLogin() {
  void router.push({
    name: 'login',
    query: { redirect: `/join/${shareCode.value}` },
  })
}

onMounted(() => {
  void loadInfo()
})
</script>

<template>
  <div class="page">
    <div class="theme-slot">
      <ThemeToggle />
    </div>
    <Card title="加入会议" class="card">
      <Alert v-if="infoError" type="error" :message="infoError" show-icon style="margin-bottom: 16px" />
      <template v-else-if="!loadingInfo && info">
        <div class="meeting-brief">
          <div class="name">{{ info.title }}</div>
          <div class="line">主持人：{{ info.hostName }}</div>
          <div v-if="!info.canJoin" class="warn">该会议已不可加入</div>
        </div>
        <Form :model="form" layout="vertical" @finish="onJoin">
          <Form.Item
            label="昵称"
            name="nickname"
            :rules="[{ required: true, whitespace: true, message: '请填写昵称' }]"
          >
            <Input v-model:value="form.nickname" placeholder="显示名称" allow-clear />
          </Form.Item>
          <Form.Item label="进房后设备">
            <div class="media-opts">
              <label class="media-opt">
                <span>麦克风</span>
                <Switch v-model:checked="form.enableMic" checked-children="开" un-checked-children="关" />
              </label>
              <label class="media-opt">
                <span>摄像头</span>
                <Switch
                  v-model:checked="form.enableCamera"
                  checked-children="开"
                  un-checked-children="关"
                />
              </label>
            </div>
          </Form.Item>
          <Button type="primary" html-type="submit" block :loading="loading" :disabled="!info.canJoin">
            进入会议
          </Button>
        </Form>
        <p v-if="loggedIn" class="hint">已登录为 {{ displayName() }}，可直接进会。</p>
        <p v-else class="hint">
          游客可填昵称进会。
          <a href="javascript:;" @click="goLogin">用账号登录</a>
        </p>
      </template>
      <p v-else class="hint">加载会议信息…</p>
    </Card>
  </div>
</template>

<style scoped>
.page {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--vc-page-grad);
  padding: 24px;
}
.theme-slot {
  position: absolute;
  top: 18px;
  right: 18px;
}
.card {
  width: 100%;
  max-width: 420px;
  box-shadow: var(--vc-shadow);
}
.meeting-brief {
  margin-bottom: 16px;
}
.meeting-brief .name {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--vc-ink);
}
.meeting-brief .line {
  color: var(--vc-muted);
  font-size: 13px;
}
.meeting-brief .warn {
  margin-top: 8px;
  color: var(--vc-danger);
  font-size: 13px;
}
.hint {
  margin-top: 16px;
  color: var(--vc-muted);
  font-size: 13px;
  line-height: 1.5;
}
.media-opts {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.media-opt {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--vc-ink-soft);
}
</style>
