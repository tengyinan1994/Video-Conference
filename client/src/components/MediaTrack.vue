<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { AttachableTrack } from '@/composables/useLiveKitRoom'

const props = defineProps<{
  track?: AttachableTrack
  muted?: boolean
  mirror?: boolean
}>()

const el = ref<HTMLMediaElement | null>(null)

async function attach() {
  // v-if 创建 video/audio 后 ref 才就绪，必须等 DOM 更新
  await nextTick()
  const media = el.value
  const track = props.track
  if (!media) return
  if (!track) {
    media.srcObject = null
    return
  }
  track.attach(media)
  media.muted = !!props.muted
  // 部分浏览器对动态挂上的流不会自动 play
  try {
    await media.play()
  } catch {
    // 自动播放策略拦截时忽略；用户手势后由浏览器恢复
  }
}

watch(
  () => [props.track, props.muted, el.value] as const,
  () => {
    void attach()
  },
  { immediate: true, flush: 'post' },
)

onBeforeUnmount(() => {
  props.track?.detach(el.value ?? undefined)
})
</script>

<template>
  <video
    v-if="track && track.kind === 'video'"
    ref="el"
    autoplay
    playsinline
    :muted="muted"
    :class="{ mirror }"
  />
  <audio
    v-else-if="track && track.kind === 'audio'"
    ref="el"
    autoplay
    :muted="muted"
  />
</template>

<style scoped>
video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: #111;
  border-radius: 12px;
}
.mirror {
  transform: scaleX(-1);
}
</style>
