<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ConfigProvider, theme as antTheme } from 'ant-design-vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { RouterView } from 'vue-router'
import { applyTheme, getTheme, subscribeTheme, type ThemeMode } from '@/stores/theme'

const mode = ref<ThemeMode>(getTheme())

const algorithm = computed(() =>
  mode.value === 'dark' ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
)

const themeToken = computed(() =>
  mode.value === 'dark'
    ? {
        colorPrimary: '#f3a04c',
        borderRadius: 12,
        colorBgContainer: '#111827',
        colorBgElevated: '#1f2937',
      }
    : {
        colorPrimary: '#f3a04c',
        borderRadius: 12,
        colorBgContainer: '#ffffff',
        colorBgElevated: '#ffffff',
      },
)

let unsub: (() => void) | undefined

onMounted(() => {
  applyTheme(getTheme())
  unsub = subscribeTheme(() => {
    mode.value = getTheme()
  })
})

onUnmounted(() => {
  unsub?.()
})
</script>

<template>
  <ConfigProvider
    :locale="zhCN"
    :theme="{
      algorithm,
      token: themeToken,
    }"
  >
    <RouterView />
  </ConfigProvider>
</template>
