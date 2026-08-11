<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Button, Checkbox, Form, Input, message } from 'ant-design-vue'
import { LockOutlined, SafetyCertificateOutlined, UserOutlined } from '@ant-design/icons-vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { fetchCaptcha, fetchLoginConfig, login } from '@/api/auth'
import { setAuth } from '@/stores/auth'
import { encryptPassword } from '@/utils/encrypt'
import { ApiError } from '@/utils/request'
import loginBg from '@/assets/images/login-bg.png'
import logoImg from '@/assets/images/logo.png'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const captchaOn = ref(true)
const captchaImg = ref('')
const autoLogin = ref(true)
const projectName = ref('视频会议系统')

const form = reactive({
  username: '',
  password: '',
  cid: '',
  code: '',
})

async function loadCaptcha() {
  try {
    const data = await fetchCaptcha()
    form.cid = data.cid
    form.code = ''
    captchaImg.value = data.base64
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '获取验证码失败')
  }
}

async function loadConfig() {
  try {
    const cfg = await fetchLoginConfig()
    captchaOn.value = cfg.captchaSwitch === 1
    if (cfg.projectName) projectName.value = cfg.projectName
  } catch {
    captchaOn.value = true
  }
  if (captchaOn.value) await loadCaptcha()
}

function handleResetPassword() {
  message.info('如果你忘记了密码，请联系管理员找回')
}

async function onSubmit() {
  if (!form.username.trim() || !form.password) {
    message.warning('请填写账号和密码')
    return
  }
  if (captchaOn.value && !form.code.trim()) {
    message.warning('请填写验证码')
    return
  }
  loading.value = true
  try {
    const data = await login({
      username: form.username.trim(),
      password: encryptPassword(form.password),
      cid: form.cid,
      code: form.code.trim(),
    })
    setAuth({
      id: data.id,
      username: data.username,
      token: data.token,
      // HotGo 返回的 expires 是有效期秒数（TTL），不是时间戳
      expiresAt: Math.floor(Date.now() / 1000) + (data.expires || 7 * 24 * 3600),
    })
    message.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    if (redirect.startsWith('/')) {
      await router.replace(redirect)
    } else {
      await router.replace({ name: 'lobby' })
    }
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '登录失败')
    if (captchaOn.value) await loadCaptcha()
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadConfig()
})
</script>

<template>
  <div class="login-page">
    <div class="login-bg" aria-hidden="true" :style="{ backgroundImage: `url(${loginBg})` }" />
    <div class="login-veil" aria-hidden="true" />

    <div class="theme-slot">
      <ThemeToggle />
    </div>

    <div class="login-shell">
      <section class="login-brand">
        <div class="brand-mark">
          <img :src="logoImg" class="brand-logo" alt="" />
          <span class="brand-kicker">Video Conference</span>
        </div>
        <h1 class="brand-title">{{ projectName }}</h1>
        <p class="brand-desc">发起或加入视频会议，与同事、访客实时协作沟通。</p>
      </section>

      <section class="login-panel">
        <div class="panel-glow" aria-hidden="true" />
        <header class="panel-header">
          <h2>欢迎回来</h2>
          <p>登录会议端，进入大厅发起或加入会议</p>
        </header>

        <Form class="login-form" :model="form" layout="vertical" @finish="onSubmit">
          <Form.Item name="username" :rules="[{ required: true, message: '请输入用户名' }]">
            <Input
              v-model:value="form.username"
              size="large"
              placeholder="请输入用户名"
              allow-clear
              autocomplete="username"
            >
              <template #prefix><UserOutlined /></template>
            </Input>
          </Form.Item>

          <Form.Item name="password" :rules="[{ required: true, message: '请输入密码' }]">
            <Input.Password
              v-model:value="form.password"
              size="large"
              placeholder="请输入密码"
              autocomplete="current-password"
            >
              <template #prefix><LockOutlined /></template>
            </Input.Password>
          </Form.Item>

          <Form.Item
            v-if="captchaOn"
            name="code"
            :rules="[{ required: true, message: '请输入验证码' }]"
          >
            <div class="captcha-row">
              <Input
                v-model:value="form.code"
                size="large"
                placeholder="验证码"
                allow-clear
              >
                <template #prefix><SafetyCertificateOutlined /></template>
              </Input>
              <img
                v-if="captchaImg"
                class="captcha-img"
                :src="captchaImg"
                alt="验证码"
                title="点击刷新"
                @click="loadCaptcha"
              />
            </div>
          </Form.Item>

          <div class="form-meta">
            <Checkbox v-model:checked="autoLogin">自动登录</Checkbox>
            <button type="button" class="link-btn" @click="handleResetPassword">忘记密码？</button>
          </div>

          <Button
            class="login-btn"
            type="primary"
            html-type="submit"
            block
            size="large"
            :loading="loading"
          >
            登录
          </Button>
        </Form>
      </section>
    </div>
  </div>
</template>

<style scoped>
.login-page {
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

.login-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  background-color: #1f2937;
  background-position: center;
  background-size: cover;
  background-repeat: no-repeat;
  transform: scale(1.02);
}

.login-veil {
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

.login-shell {
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

.login-brand {
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

.login-panel {
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
  line-height: 1.3;
}

.panel-header p {
  margin: 8px 0 0;
  color: var(--login-ink-mute);
  font-size: 13px;
  line-height: 1.55;
}

.login-form :deep(.ant-form-item) {
  margin-bottom: 18px;
}

.login-form :deep(.ant-form-item-explain-error) {
  color: #fda4af;
}

.login-form :deep(.ant-input-affix-wrapper),
.login-form :deep(.ant-input-affix-wrapper-lg),
.login-form :deep(.ant-input),
.login-form :deep(.ant-input-lg) {
  height: 40px;
  color: rgba(248, 250, 252, 0.95) !important;
  background: var(--login-field) !important;
  border: 1px solid var(--login-field-border) !important;
  border-radius: 10px !important;
  box-shadow: none !important;
}

.login-form :deep(.ant-input-affix-wrapper > input.ant-input) {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  height: auto;
}

.login-form :deep(.ant-input-affix-wrapper:hover),
.login-form :deep(.ant-input-affix-wrapper-focused),
.login-form :deep(.ant-input-outlined:hover),
.login-form :deep(.ant-input-outlined:focus),
.login-form :deep(.ant-input:hover),
.login-form :deep(.ant-input:focus) {
  border-color: rgba(243, 160, 76, 0.7) !important;
  background: rgba(255, 255, 255, 0.14) !important;
}

.login-form :deep(.ant-input::placeholder),
.login-form :deep(input::placeholder) {
  color: rgba(203, 213, 225, 0.55) !important;
}

.login-form :deep(.ant-input-prefix) {
  color: rgba(226, 232, 240, 0.72);
  margin-inline-end: 8px;
}

.login-form :deep(.ant-input-password-icon),
.login-form :deep(.ant-input-clear-icon) {
  color: rgba(226, 232, 240, 0.72) !important;
}

.captcha-row {
  display: flex;
  gap: 10px;
  align-items: center;
  width: 100%;
}

.captcha-row :deep(.ant-input-affix-wrapper) {
  flex: 1;
  min-width: 0;
}

.captcha-img {
  width: 100px;
  height: 40px;
  object-fit: cover;
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.16);
  cursor: pointer;
  flex-shrink: 0;
  /* 压低库默认的高饱和随机配色，贴近玻璃面板 */
  filter: saturate(0.35) contrast(1.08) brightness(1.05);
}

.form-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 2px 0 24px;
}

.form-meta :deep(.ant-checkbox-wrapper) {
  color: var(--login-ink-soft);
}

.form-meta :deep(.ant-checkbox-inner) {
  background: transparent;
  border-color: rgba(255, 255, 255, 0.28);
}

.form-meta :deep(.ant-checkbox-checked .ant-checkbox-inner) {
  background: var(--login-brand);
  border-color: var(--login-brand);
}

.link-btn {
  padding: 0;
  border: 0;
  background: transparent;
  color: rgba(226, 232, 240, 0.78);
  font-size: 14px;
  cursor: pointer;
}

.link-btn:hover {
  color: #fff;
}

.login-btn {
  height: 44px !important;
  border-radius: 10px !important;
  border-color: var(--login-brand) !important;
  background: var(--login-brand) !important;
  color: #fff !important;
  font-weight: 650 !important;
  letter-spacing: 0.04em;
  box-shadow: 0 12px 28px rgba(243, 160, 76, 0.32) !important;
}

.login-btn:hover {
  border-color: var(--login-brand-strong) !important;
  background: var(--login-brand-strong) !important;
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
  .login-shell {
    grid-template-columns: 1fr;
    align-content: center;
    gap: 28px;
    padding-top: 64px;
    padding-bottom: 48px;
  }

  .login-brand {
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
