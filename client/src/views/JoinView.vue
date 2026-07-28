<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Button, Card, Form, Input, Switch, message } from 'ant-design-vue'
import { createToken } from '@/api/conference'
import { ApiError } from '@/utils/request'

const router = useRouter()
const loading = ref(false)
const form = reactive({
  room: 'demo',
  nickname: '',
  enableMic: false,
  enableCamera: false,
})

async function onJoin() {
  if (!form.room.trim() || !form.nickname.trim()) {
    message.warning('请填写房间名和昵称')
    return
  }
  loading.value = true
  try {
    const data = await createToken(form.room.trim(), form.nickname.trim())
    sessionStorage.setItem(
      'vc.session',
      JSON.stringify({
        serverUrl: data.serverUrl,
        token: data.token,
        room: data.room,
        identity: data.identity,
        nickname: data.nickname,
        expiresAt: data.expiresAt,
        isHost: !!data.isHost,
        enableMic: form.enableMic,
        enableCamera: form.enableCamera,
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
</script>

<template>
  <div class="page">
    <Card title="加入会议" class="card">
      <Form :model="form" layout="vertical" @finish="onJoin">
        <Form.Item
          label="房间名"
          name="room"
          :rules="[{ required: true, whitespace: true, message: '请填写房间名' }]"
        >
          <Input v-model:value="form.room" placeholder="例如 demo" allow-clear />
        </Form.Item>
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
        <Button type="primary" html-type="submit" block :loading="loading">进入会议</Button>
      </Form>
      <p class="hint">默认静音且关闭摄像头；可在进房前自行打开。进房后仍可随时切换。</p>
    </Card>
  </div>
</template>

<style scoped>
.page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(160deg, #0f172a, #1e293b 45%, #0ea5e9);
  padding: 24px;
}
.card {
  width: 100%;
  max-width: 420px;
}
.hint {
  margin-top: 16px;
  color: #64748b;
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
  color: rgba(0, 0, 0, 0.88);
}
</style>
