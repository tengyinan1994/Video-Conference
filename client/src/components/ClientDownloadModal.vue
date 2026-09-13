<script setup lang="ts">
import { ref } from 'vue'
import { Button, Modal, message } from 'ant-design-vue'
import {
  AndroidOutlined,
  AppleOutlined,
  DesktopOutlined,
  DownloadOutlined,
  WindowsOutlined,
} from '@ant-design/icons-vue'
import { downloadClientInstaller } from '@/api/conference'
import { ApiError } from '@/utils/request'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const downloading = ref(false)

const platforms = [
  {
    id: 'windows',
    label: 'Windows',
    available: true,
    hint: '安装时请信任证书（安装包会写入当前用户根证书；若系统弹出安全提示请允许）。',
  },
  { id: 'macos', label: 'macOS', available: false },
  { id: 'linux', label: 'Linux', available: false },
  { id: 'ios', label: 'iOS', available: false },
  { id: 'android', label: 'Android', available: false },
] as const

function platformIcon(id: string) {
  if (id === 'windows') return WindowsOutlined
  if (id === 'macos' || id === 'ios') return AppleOutlined
  if (id === 'android') return AndroidOutlined
  return DesktopOutlined
}

async function onDownload(platform: string) {
  if (platform !== 'windows') return
  downloading.value = true
  const hide = message.loading('正在下载安装包…', 0)
  try {
    await downloadClientInstaller(platform)
    message.success('已开始下载')
  } catch (err) {
    message.error(err instanceof ApiError ? err.message : '下载失败')
  } finally {
    hide()
    downloading.value = false
  }
}
</script>

<template>
  <Modal
    :open="open"
    title="下载客户端"
    :footer="null"
    wrap-class-name="client-download-modal"
    destroy-on-close
    @update:open="emit('update:open', $event)"
  >
    <ul class="platform-list">
      <li v-for="item in platforms" :key="item.id" class="platform-row">
        <div class="platform-meta">
          <div class="platform-title">
            <component :is="platformIcon(item.id)" class="platform-icon" />
            <span>{{ item.label }}</span>
          </div>
          <p v-if="item.available && item.hint" class="platform-hint">{{ item.hint }}</p>
          <p v-else class="platform-soon">敬请期待</p>
        </div>
        <Button
          v-if="item.available"
          type="primary"
          class="btn-download"
          :loading="downloading"
          @click="onDownload(item.id)"
        >
          <template #icon><DownloadOutlined /></template>
          下载
        </Button>
        <Button v-else disabled class="btn-soon">敬请期待</Button>
      </li>
    </ul>
  </Modal>
</template>

<style scoped>
.platform-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
  padding: 4px 0 2px;
  list-style: none;
}

.platform-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
  border: 1px solid var(--line, rgba(15, 23, 42, 0.08));
  border-radius: 12px;
  background: color-mix(in srgb, var(--card, #fff) 88%, transparent);
}

.platform-meta {
  min-width: 0;
  flex: 1;
}

.platform-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 650;
  color: var(--ink, #0f172a);
}

.platform-icon {
  font-size: 16px;
  color: var(--ink-60, #64748b);
}

.platform-hint,
.platform-soon {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.55;
}

.platform-hint {
  color: var(--ink-60, #64748b);
}

.platform-soon {
  color: var(--ink-60, #64748b);
}

.btn-download {
  flex-shrink: 0;
  border-radius: 10px !important;
}

.btn-soon {
  flex-shrink: 0;
  border-radius: 10px !important;
}
</style>

<style>
.client-download-modal .ant-modal-content {
  overflow: hidden;
  border-radius: 14px;
}

.client-download-modal .ant-modal-header {
  margin: 0;
  padding: 20px 24px 8px;
  border-bottom: none;
}

.client-download-modal .ant-modal-title {
  color: var(--vc-ink);
  font-size: 17px;
  font-weight: 650;
  line-height: 1.35;
}

.client-download-modal .ant-modal-close {
  top: 16px;
}

.client-download-modal .ant-modal-body {
  padding: 4px 24px 22px;
}
</style>
