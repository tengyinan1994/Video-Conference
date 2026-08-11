<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button, Form, Input, Switch, message } from 'ant-design-vue'
import { UserOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { createToken, shareView, type MeetingShareView } from '@/api/conference'
import { displayName, isLoggedIn } from '@/stores/auth'
import { ApiError } from '@/utils/request'
import loginBg from '@/assets/images/login-bg.png'
import logoImg from '@/assets/images/logo.png'

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

const meetingTime = computed(() => {
  const m = info.value
  if (!m?.startAt) return ''
  const start = dayjs(m.startAt)
  const end = m.endAt ? dayjs(m.endAt) : null
  const weekdays = ['日', '一', '二', '三', '四', '五', '六']
  const date = `${start.format('YYYY年M月D日')} 周${weekdays[start.day()]}`
  const range = end ? `${start.format('HH:mm')} – ${end.format('HH:mm')}` : start.format('HH:mm')
  return `${date} ${range}`
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
  <div class="join-page">
    <div class="join-bg" aria-hidden="true" :style="{ backgroundImage: `url(${loginBg})` }" />
    <div class="join-veil" aria-hidden="true" />

    <div class="theme-slot">
      <ThemeToggle />
    </div>

    <div class="join-shell">
      <section class="join-brand">
        <div class="brand-mark">
          <img :src="logoImg" class="brand-logo" alt="" />
          <span class="brand-kicker">Video Conference</span>
        </div>
        <h1 class="brand-title">加入会议</h1>
        <p class="brand-desc">打开分享链接即可入会。游客填写昵称，已有账号也可直接进入。</p>
      </section>

      <section class="join-panel">
        <div class="panel-glow" aria-hidden="true" />

        <div v-if="infoError" class="state-box state-error">{{ infoError }}</div>
        <div v-else-if="loadingInfo" class="state-box">加载会议信息…</div>

        <template v-else-if="info">
          <header class="panel-header">
            <h2>{{ info.title }}</h2>
            <p>
              <span class="meta-label">主持人</span>
              {{ info.hostName }}
            </p>
            <p v-if="meetingTime" class="meta-time">{{ meetingTime }}</p>
            <p v-if="!info.canJoin" class="meta-warn">该会议已不可加入</p>
          </header>

          <Form class="join-form" :model="form" layout="vertical" @finish="onJoin">
            <Form.Item
              label="昵称"
              name="nickname"
              required
              :rules="[{ required: true, whitespace: true, message: '请填写昵称' }]"
            >
              <Input
                v-model:value="form.nickname"
                size="large"
                placeholder="请输入显示昵称"
                allow-clear
              >
                <template #prefix><UserOutlined /></template>
              </Input>
            </Form.Item>

            <div class="media-opts">
              <div class="media-opt">
                <span>麦克风</span>
                <Switch
                  v-model:checked="form.enableMic"
                  checked-children="开"
                  un-checked-children="关"
                />
              </div>
              <div class="media-opt">
                <span>摄像头</span>
                <Switch
                  v-model:checked="form.enableCamera"
                  checked-children="开"
                  un-checked-children="关"
                />
              </div>
            </div>

            <Button
              class="join-btn"
              type="primary"
              html-type="submit"
              block
              size="large"
              :loading="loading"
              :disabled="!info.canJoin"
            >
              进入会议
            </Button>
          </Form>

          <p v-if="loggedIn" class="hint">已登录为 {{ displayName() }}，可直接进会。</p>
          <p v-else class="hint">
            游客可填昵称进会。
            <button type="button" class="link-btn" @click="goLogin">用账号登录</button>
          </p>
        </template>
      </section>
    </div>
  </div>
</template>

<style scoped>
.join-page {
  --login-brand: #f3a04c;
  --login-brand-strong: #e8892a;
  --login-ink: rgba(248, 250, 252, 0.96);
  --login-ink-soft: rgba(226, 232, 240, 0.78);
  --login-ink-mute: rgba(203, 213, 225, 0.62);
  --login-glass-border: rgba(255, 255, 255, 0.16);
  --login-field: rgba(255, 255, 255, 0.1);
  --login-field-border: rgba(255, 255, 255, 0.18);

  position: relative;
  display: flex;
  min-height: 100vh;
  overflow: auto;
  color: var(--login-ink);
}

.join-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  background-color: #1f2937;
  background-position: center;
  background-size: cover;
  background-repeat: no-repeat;
  transform: scale(1.02);
}

.join-veil {
  position: fixed;
  inset: 0;
  z-index: 1;
  pointer-events: none;
  background:
    linear-gradient(
      115deg,
      rgba(8, 12, 22, 0.78) 0%,
      rgba(8, 12, 22, 0.58) 34%,
      rgba(8, 12, 22, 0.28) 62%,
      rgba(8, 12, 22, 0.18) 100%
    ),
    radial-gradient(900px 520px at 12% 18%, rgba(243, 160, 76, 0.22), transparent 58%),
    radial-gradient(780px 480px at 88% 82%, rgba(59, 130, 246, 0.16), transparent 55%);
}

.theme-slot {
  position: fixed;
  top: 18px;
  right: 18px;
  z-index: 3;
}

.join-shell {
  position: relative;
  z-index: 2;
  display: grid;
  grid-template-columns: minmax(280px, 1.05fr) minmax(320px, 420px);
  align-items: center;
  gap: clamp(28px, 6vw, 88px);
  width: 100%;
  min-height: 100vh;
  padding: clamp(28px, 5vw, 72px) clamp(20px, 7vw, 96px);
}

.join-brand {
  max-width: 560px;
  animation: rise-in 0.7s ease both;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 22px;
}

.brand-logo {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  object-fit: cover;
  box-shadow: 0 10px 28px rgba(243, 160, 76, 0.35);
}

.brand-kicker {
  color: var(--login-brand);
  font-size: 12px;
  font-weight: 650;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.brand-title {
  margin: 0;
  color: var(--login-ink);
  font-size: clamp(36px, 5vw, 54px);
  font-weight: 720;
  line-height: 1.12;
  letter-spacing: 0.01em;
  text-shadow: 0 10px 40px rgba(0, 0, 0, 0.35);
}

.brand-desc {
  margin: 18px 0 0;
  max-width: 34em;
  color: var(--login-ink-soft);
  font-size: 16px;
  line-height: 1.7;
}

.join-panel {
  position: relative;
  width: 100%;
  padding: 28px 26px 26px;
  border-radius: 22px;
  border: 1px solid var(--login-glass-border);
  background: linear-gradient(160deg, rgba(255, 255, 255, 0.14), rgba(15, 23, 42, 0.28));
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.12) inset,
    0 24px 60px rgba(0, 0, 0, 0.28);
  backdrop-filter: blur(22px) saturate(1.2);
  -webkit-backdrop-filter: blur(22px) saturate(1.2);
  animation: rise-in 0.75s ease 0.08s both;
}

.panel-glow {
  position: absolute;
  top: -1px;
  left: 18%;
  right: 18%;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(243, 160, 76, 0.75), transparent);
  opacity: 0.9;
}

.panel-header {
  margin-bottom: 22px;
}

.panel-header h2 {
  margin: 0;
  color: var(--login-ink);
  font-size: 22px;
  font-weight: 650;
  line-height: 1.35;
}

.panel-header p {
  margin: 10px 0 0;
  color: var(--login-ink-mute);
  font-size: 13px;
  line-height: 1.55;
}

.meta-label {
  position: relative;
  padding-right: 8px;
  margin-right: 2px;
  color: var(--login-brand);
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
  background: rgba(255, 255, 255, 0.22);
}

.meta-time {
  font-variant-numeric: tabular-nums;
}

.meta-warn {
  color: #fda4af !important;
}

.state-box {
  padding: 18px 4px;
  color: var(--login-ink-mute);
  font-size: 14px;
  line-height: 1.6;
}

.state-error {
  color: #fda4af;
}

.join-form :deep(.ant-form-item) {
  margin-bottom: 18px;
}

.join-form :deep(.ant-form-item-label > label) {
  color: var(--login-ink-soft);
}

.join-form :deep(.ant-form-item-required::before) {
  color: #fb7185 !important;
}

.join-form :deep(.ant-form-item-explain-error) {
  color: #fda4af;
}

.join-form :deep(.ant-input-affix-wrapper),
.join-form :deep(.ant-input-affix-wrapper-lg),
.join-form :deep(.ant-input),
.join-form :deep(.ant-input-lg) {
  height: 40px;
  color: rgba(248, 250, 252, 0.95) !important;
  background: var(--login-field) !important;
  border: 1px solid var(--login-field-border) !important;
  border-radius: 10px !important;
  box-shadow: none !important;
}

.join-form :deep(.ant-input-affix-wrapper > input.ant-input) {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  height: auto;
}

.join-form :deep(.ant-input-affix-wrapper:hover),
.join-form :deep(.ant-input-affix-wrapper-focused),
.join-form :deep(.ant-input:hover),
.join-form :deep(.ant-input:focus) {
  border-color: rgba(243, 160, 76, 0.7) !important;
  background: rgba(255, 255, 255, 0.14) !important;
}

.join-form :deep(.ant-input::placeholder),
.join-form :deep(input::placeholder) {
  color: rgba(203, 213, 225, 0.55) !important;
}

.join-form :deep(.ant-input-prefix) {
  color: rgba(226, 232, 240, 0.72);
  margin-inline-end: 8px;
}

.join-form :deep(.ant-input-clear-icon) {
  color: rgba(226, 232, 240, 0.72) !important;
}

.media-opts {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 24px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
}

.media-opt {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--login-ink-soft);
  font-size: 14px;
}

.media-opt :deep(.ant-switch) {
  background: rgba(148, 163, 184, 0.35);
}

.media-opt :deep(.ant-switch-checked) {
  background: var(--login-brand) !important;
}

.join-btn {
  height: 44px !important;
  border-radius: 10px !important;
  border-color: var(--login-brand) !important;
  background: var(--login-brand) !important;
  color: #fff !important;
  font-weight: 650 !important;
  letter-spacing: 0.04em;
  box-shadow: 0 12px 28px rgba(243, 160, 76, 0.32) !important;
}

.join-btn:hover {
  border-color: var(--login-brand-strong) !important;
  background: var(--login-brand-strong) !important;
}

.join-btn:disabled {
  opacity: 0.55 !important;
}

.hint {
  margin: 16px 0 0;
  color: var(--login-ink-mute);
  font-size: 12px;
  line-height: 1.55;
}

.link-btn {
  padding: 0;
  border: 0;
  background: transparent;
  color: #93c5fd;
  font-size: inherit;
  cursor: pointer;
}

.link-btn:hover {
  color: #bfdbfe;
  text-decoration: underline;
}

@keyframes rise-in {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 960px) {
  .join-shell {
    grid-template-columns: 1fr;
    align-content: center;
    gap: 28px;
    padding-top: 64px;
    padding-bottom: 48px;
  }

  .join-brand {
    max-width: none;
  }

  .brand-title {
    font-size: clamp(30px, 8vw, 40px);
  }

  .brand-desc {
    font-size: 14px;
  }
}
</style>
