<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Button, Tooltip } from 'ant-design-vue'
import { getTheme, subscribeTheme, toggleTheme, type ThemeMode } from '@/stores/theme'

const mode = ref<ThemeMode>(getTheme())
const isDark = computed(() => mode.value === 'dark')
const tip = computed(() => (isDark.value ? '切换到浅色' : '切换到深色'))

let unsub: (() => void) | undefined

onMounted(() => {
  unsub = subscribeTheme(() => {
    mode.value = getTheme()
  })
})

onUnmounted(() => {
  unsub?.()
})
</script>

<template>
  <Tooltip :title="tip">
    <Button type="text" class="theme-toggle" @click="toggleTheme">
      <template #icon>
        <span class="theme-icon-wrap" aria-hidden="true">
          <svg
            v-if="isDark"
            class="theme-icon"
            viewBox="0 0 24 24"
            width="1em"
            height="1em"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8z" />
          </svg>
          <svg
            v-else
            class="theme-icon"
            viewBox="0 0 24 24"
            width="1em"
            height="1em"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <circle cx="12" cy="12" r="4" />
            <path
              d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"
            />
          </svg>
        </span>
      </template>
    </Button>
  </Tooltip>
</template>

<style scoped>
.theme-toggle {
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  width: 32px !important;
  height: 32px !important;
  min-width: 32px !important;
  padding: 0 !important;
  line-height: 1 !important;
  color: var(--vc-ink-soft, var(--ink-60, #64748b)) !important;
  border: none !important;
  background: transparent !important;
  box-shadow: none !important;
}

.theme-toggle:hover,
.theme-toggle:focus {
  color: var(--vc-ink, var(--ink, #0f172a)) !important;
  background: transparent !important;
}

.theme-toggle :deep(.ant-btn-icon) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin: 0 !important;
  line-height: 0;
}

.theme-icon-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  line-height: 0;
}

.theme-icon {
  display: block;
  width: 18px;
  height: 18px;
  vertical-align: middle;
}
</style>
