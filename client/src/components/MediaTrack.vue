<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { AttachableTrack } from '@/composables/useLiveKitRoom'

const props = defineProps<{
  track?: AttachableTrack
  muted?: boolean
  mirror?: boolean
  /** cover=人像；contain=投屏，避免桌面被裁切 */
  fit?: 'cover' | 'contain'
}>()

const el = ref<HTMLMediaElement | null>(null)
let endedHandler: (() => void) | null = null
let boundMediaTrack: MediaStreamTrack | null = null
/** 当前挂在本元素上的 LiveKit Track；换轨时必须先 detach，否则 unmute/restart 会把旧轨重新贴回主视图 */
let attachedTrack: AttachableTrack | null = null
let attachedMedia: HTMLMediaElement | null = null
let attachGen = 0
let disposed = false

function clearEndedListener() {
  if (boundMediaTrack && endedHandler) {
    boundMediaTrack.removeEventListener('ended', endedHandler)
  }
  boundMediaTrack = null
  endedHandler = null
}

function detachCurrent() {
  const track = attachedTrack
  const media = attachedMedia
  attachedTrack = null
  attachedMedia = null
  if (!track || !media) return
  try {
    // 必须带 element 调用 detach：无参 detach 会卸掉该轨在侧栏等其它元素上的画面
    track.detach(media)
  } catch {
    // 轨已销毁时忽略
  }
  if (media.srcObject) media.srcObject = null
}

async function attach() {
  const gen = ++attachGen
  const requested = props.track
  // v-if 创建 video/audio 后 ref 才就绪，必须等 DOM 更新
  await nextTick()
  if (disposed || gen !== attachGen) return
  const media = el.value
  clearEndedListener()

  if (attachedTrack && attachedTrack !== requested) {
    detachCurrent()
  }

  if (!media) return
  if (!requested) {
    detachCurrent()
    media.srcObject = null
    return
  }
  // 已结束的轨不要再挂载，避免黑屏残留
  if (requested.mediaStreamTrack && requested.mediaStreamTrack.readyState !== 'live') {
    detachCurrent()
    media.srcObject = null
    return
  }
  requested.attach(media)
  attachedTrack = requested
  attachedMedia = media
  media.muted = !!props.muted
  const liveTrack = requested.mediaStreamTrack
  if (liveTrack) {
    boundMediaTrack = liveTrack
    endedHandler = () => {
      const elMedia = el.value
      if (!elMedia || !(elMedia.srcObject instanceof MediaStream)) return
      // 轨 restart 后 LiveKit 会换上新 MST；旧轨 ended 时若已不在 srcObject 里，不要清掉新画面
      if (!elMedia.srcObject.getTracks().includes(liveTrack)) return
      detachCurrent()
    }
    liveTrack.addEventListener('ended', endedHandler)
  }
  // 部分浏览器对动态挂上的流不会自动 play
  try {
    await media.play()
  } catch {
    // 自动播放策略拦截时忽略；用户手势后由浏览器恢复
  }
  if (disposed || gen !== attachGen) {
    if (attachedTrack === requested && attachedMedia === media) detachCurrent()
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
  disposed = true
  attachGen += 1
  clearEndedListener()
  detachCurrent()
})
</script>

<template>
  <video
    v-if="track && track.kind === 'video'"
    ref="el"
    autoplay
    playsinline
    :muted="muted"
    :class="{ mirror, contain: fit === 'contain' }"
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
.contain {
  object-fit: contain;
  background: #000;
}
</style>
