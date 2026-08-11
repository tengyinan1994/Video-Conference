<template>
  <div class="view-account">
    <div
      class="view-account-bg"
      aria-hidden="true"
      :style="{ backgroundImage: `url(${loginBg})` }"
    />
    <div class="view-account-veil" aria-hidden="true" />

    <div class="view-account-shell">
      <section class="view-account-brand">
        <div class="brand-mark">
          <img src="~@/assets/images/logo.png" class="account-logo" alt="" />
          <span class="brand-kicker">Video Conference</span>
        </div>
        <h1 class="brand-title">{{ projectName || '视频会议系统' }}</h1>
        <p class="brand-desc">统一管理会议人员与身份认证，保障每一次协作安全可控。</p>
      </section>

      <section class="view-account-panel">
        <div class="panel-glow" aria-hidden="true" />
        <header class="panel-header">
          <h2>欢迎回来</h2>
          <p>登录管理后台，继续组织与运维会议</p>
        </header>
        <main class="view-account-main">
          <transition name="fade-slide" appear>
            <component
              :is="activeModule.component"
              @updateActiveModule="handleUpdateActiveModule"
            />
          </transition>
        </main>
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import LoginFrom from './login/index.vue';
  import RegisterFrom from './register/index.vue';
  import { useRouter } from 'vue-router';
  import { useUserStore } from '@/store/modules/user';
  import loginBg from '@/assets/images/login-bg.png';

  const userStore = useUserStore();
  const projectName = computed(() => userStore.loginConfig?.projectName);

  interface LoginModule {
    key: string;
    label: string;
    component: Component;
  }

  const router = useRouter();
  const activeModule = ref<LoginModule>({
    key: 'login',
    label: '账号登录',
    component: LoginFrom,
  });

  const modules: LoginModule[] = [{ key: 'login', label: '账号登录', component: LoginFrom }];

  function handleUpdateActiveModule(key: string) {
    const findItem = modules.find((item) => item.key === key);
    if (findItem) {
      activeModule.value = findItem;
    }
  }

  onMounted(() => {
    if (userStore.loginConfig?.loginRegisterSwitch === 1) {
      const findItem = modules.find((item) => item.key === 'register');
      if (!findItem) {
        modules.push({ key: 'register', label: '注册账号', component: RegisterFrom });
      }
    }

    const key = router.currentRoute.value.query?.scope as string;
    if (key) {
      handleUpdateActiveModule(key);
    }
  });
</script>

<style lang="less" scoped>
  .view-account {
    --login-brand: #f3a04c;
    --login-brand-strong: #e8892a;
    --login-ink: rgba(248, 250, 252, 0.96);
    --login-ink-soft: rgba(226, 232, 240, 0.78);
    --login-ink-mute: rgba(203, 213, 225, 0.62);
    --login-glass: rgba(15, 23, 42, 0.42);
    --login-glass-border: rgba(255, 255, 255, 0.16);
    --login-field: rgba(255, 255, 255, 0.1);
    --login-field-border: rgba(255, 255, 255, 0.18);

    position: relative;
    display: flex;
    min-height: 100vh;
    overflow: auto;
    color: var(--login-ink);
  }

  .view-account-bg {
    position: fixed;
    inset: 0;
    z-index: 0;
    background-color: #1f2937;
    background-position: center;
    background-size: cover;
    background-repeat: no-repeat;
    transform: scale(1.02);
  }

  .view-account-veil {
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

  .view-account-shell {
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

  .view-account-brand {
    max-width: 560px;
    animation: rise-in 0.7s ease both;
  }

  .brand-mark {
    display: inline-flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 22px;
  }

  .account-logo {
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

  .view-account-panel {
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

  .view-account-main {
    width: 100%;
  }

  :deep(.n-form-item) {
    margin-bottom: 18px;
  }

  :deep(.n-input) {
    --n-border: 1px solid var(--login-field-border) !important;
    --n-border-hover: 1px solid rgba(243, 160, 76, 0.55) !important;
    --n-border-focus: 1px solid rgba(243, 160, 76, 0.8) !important;
    --n-color: var(--login-field) !important;
    --n-color-focus: rgba(255, 255, 255, 0.14) !important;
    --n-text-color: rgba(248, 250, 252, 0.95) !important;
    --n-placeholder-color: rgba(203, 213, 225, 0.55) !important;
    --n-caret-color: var(--login-brand) !important;
    --n-box-shadow: none !important;
    backdrop-filter: blur(8px);
  }

  :deep(.n-input .n-input__input-el),
  :deep(.n-input .n-input__textarea-el) {
    color: rgba(248, 250, 252, 0.95);
  }

  :deep(.n-input .n-icon) {
    color: rgba(226, 232, 240, 0.72) !important;
  }

  :deep(.n-checkbox .n-checkbox__label) {
    color: var(--login-ink-soft);
  }

  :deep(.n-checkbox .n-checkbox-box) {
    --n-border: 1px solid rgba(255, 255, 255, 0.28);
    --n-border-checked: var(--login-brand);
    --n-color-checked: var(--login-brand);
  }

  :deep(.n-button:not(.n-button--primary-type)) {
    color: rgba(226, 232, 240, 0.78) !important;
  }

  :deep(.n-button--primary-type) {
    --n-color: var(--login-brand) !important;
    --n-color-hover: var(--login-brand-strong) !important;
    --n-color-pressed: #d97818 !important;
    --n-color-focus: var(--login-brand) !important;
    --n-border: 1px solid var(--login-brand) !important;
    --n-border-hover: 1px solid var(--login-brand-strong) !important;
    --n-border-pressed: 1px solid #d97818 !important;
    --n-border-focus: 1px solid var(--login-brand) !important;
    --n-text-color: #fff !important;
    --n-text-color-hover: #fff !important;
    --n-text-color-pressed: #fff !important;
    --n-text-color-focus: #fff !important;
    height: 44px !important;
    font-weight: 650 !important;
    letter-spacing: 0.04em;
    box-shadow: 0 12px 28px rgba(243, 160, 76, 0.32) !important;
  }

  :deep(.n-input-group img),
  :deep(.captcha-img) {
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid rgba(255, 255, 255, 0.16);
    filter: saturate(0.35) contrast(1.08) brightness(1.05);
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
    .view-account-shell {
      grid-template-columns: 1fr;
      align-content: center;
      gap: 28px;
      padding-top: 48px;
      padding-bottom: 48px;
    }

    .view-account-brand {
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
