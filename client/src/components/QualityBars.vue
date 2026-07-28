<script setup lang="ts">
import { computed } from 'vue'
import type { QualityLevel } from '@/composables/useLiveKitRoom'

const props = defineProps<{
  quality: QualityLevel
}>()

const bars = computed(() => {
  switch (props.quality) {
    case 'excellent':
      return 4
    case 'good':
      return 3
    case 'poor':
      return 2
    case 'lost':
      return 1
    default:
      return 0
  }
})

const title = computed(() => {
  switch (props.quality) {
    case 'excellent':
      return '网络优秀'
    case 'good':
      return '网络良好'
    case 'poor':
      return '网络较差'
    case 'lost':
      return '网络中断'
    default:
      return '网络未知'
  }
})
</script>

<template>
  <span class="bars" :title="title" aria-hidden="true">
    <i v-for="n in 4" :key="n" :class="{ on: n <= bars, warn: bars <= 2 && n <= bars }" />
  </span>
</template>

<style scoped>
.bars {
  display: inline-flex;
  align-items: flex-end;
  gap: 2px;
  height: 12px;
  margin-left: 6px;
  vertical-align: middle;
}
.bars i {
  display: block;
  width: 3px;
  border-radius: 1px;
  background: #475569;
}
.bars i:nth-child(1) {
  height: 3px;
}
.bars i:nth-child(2) {
  height: 6px;
}
.bars i:nth-child(3) {
  height: 9px;
}
.bars i:nth-child(4) {
  height: 12px;
}
.bars i.on {
  background: #22c55e;
}
.bars i.warn {
  background: #f59e0b;
}
</style>
