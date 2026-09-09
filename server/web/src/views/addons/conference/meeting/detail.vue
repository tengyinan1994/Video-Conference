<template>
  <div>
    <n-modal
      v-model:show="showModal"
      :mask-closable="true"
      :show-icon="false"
      preset="dialog"
      transform-origin="center"
      :title="`会议详情${formValue.id > 0 ? ' #' + formValue.id : ''}`"
      :style="{ width: dialogWidth }"
    >
      <n-scrollbar style="max-height: 82vh" class="pr-5">
        <n-spin :show="loading" description="请稍候...">
          <n-descriptions
            bordered
            :column="2"
            size="small"
            label-placement="left"
            :label-style="{ width: '88px' }"
          >
            <n-descriptions-item label="会议名称" :span="2">
              {{ formValue.title }}
            </n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="statusTagType" size="small">
                {{ dict.getLabel('MeetingStatusOptions', formValue.status) || formValue.status }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="主持人">{{ formValue.hostName }}</n-descriptions-item>
            <n-descriptions-item label="会议类型" :span="2">{{
              formValue.typeName || '未分类'
            }}</n-descriptions-item>
            <n-descriptions-item label="会议时间" :span="2">{{ timeText }}</n-descriptions-item>
            <n-descriptions-item label="创建时间" :span="2">{{
              fmtTime(formValue.createdAt)
            }}</n-descriptions-item>
            <n-descriptions-item label="录制开关" :span="2">
              {{ formValue.recordEnabled ? '已开启' : '未开启' }}
            </n-descriptions-item>
          </n-descriptions>

          <div class="detail-section">
            <div class="detail-section-title">参会人员（{{ attendeeList.length }}）</div>
            <template v-if="attendeeList.length">
              <div class="attendee-tags">
                <n-tag v-for="name in attendeeList" :key="name" size="small" class="attendee-tag">
                  {{ name }}
                </n-tag>
              </div>
            </template>
            <div v-else class="detail-empty">暂无参会人员</div>
          </div>

          <div class="detail-section">
            <div class="detail-section-title">录制（{{ recordings.length }}）</div>
            <template v-if="recordings.length">
              <div v-for="seg in recordings" :key="seg.id" class="recording-item">
                <span class="recording-name">第{{ seg.seq }}段</span>
                <template v-if="seg.status === 'complete' && seg.id">
                  <a
                    :href="recordingProxyUrl('play', seg.id)"
                    target="_blank"
                    rel="noopener"
                    class="recording-link"
                  >
                    回放
                  </a>
                  <a
                    :href="recordingProxyUrl('download', seg.id)"
                    :download="`recording-${formValue.id}-${seg.seq}.mp4`"
                    rel="noopener"
                    class="recording-link"
                  >
                    下载
                  </a>
                </template>
                <span v-else class="recording-status">{{ recordingStatusText(seg.status) }}</span>
              </div>
            </template>
            <div v-else-if="formValue.recordEnabled" class="detail-empty"
              >已开启录制（暂无文件）</div
            >
            <div v-else class="detail-empty">未开启录制</div>
          </div>
        </n-spin>
      </n-scrollbar>
    </n-modal>
  </div>
</template>

<script lang="ts" setup>
  import { ref, computed } from 'vue';
  import { View } from '@/api/addons/conference/meeting';
  import { recordingProxyUrl } from '@/api/addons/conference/meeting';
  import { State, newState } from './model';
  import { useDictStore } from '@/store/modules/dict';
  import { adaModalWidth } from '@/utils/hotgo';
  import { formatToDateTime } from '@/utils/dateUtil';

  const dict = useDictStore();
  const showModal = ref(false);
  const loading = ref(false);
  const formValue = ref<State>(newState(null));
  const dialogWidth = computed(() => adaModalWidth(660));

  const statusTagType = computed(() => {
    const map: Record<string, string> = {
      scheduled: 'info',
      ongoing: 'success',
      ended: 'default',
    };
    return map[formValue.value.status] || 'default';
  });

  const attendeeList = computed<string[]>(() =>
    Array.isArray(formValue.value.attendees) ? formValue.value.attendees : []
  );

  const recordings = computed(() =>
    Array.isArray(formValue.value.recordings) ? formValue.value.recordings : []
  );

  const timeText = computed(() => {
    const s = fmtTime(formValue.value.startAt);
    const e = fmtTime(formValue.value.endAt);
    if (!s || !e) return '-';
    return `${s} 至 ${e}`;
  });

  function fmtTime(v: string | number | null): string {
    if (v === null || v === undefined || v === '') return '';
    return formatToDateTime(String(v));
  }

  function recordingStatusText(status: string): string {
    switch (status) {
      case 'starting':
        return '开始中';
      case 'active':
        return '录制中';
      case 'stopping':
        return '停止中';
      case 'complete':
        return '无文件';
      case 'failed':
        return '失败';
      default:
        return status || '处理中';
    }
  }

  function openModal(record: Recordable) {
    showModal.value = true;
    if (!record || record.id < 1) return;
    loading.value = true;
    View({ id: record.id })
      .then((res) => {
        formValue.value = newState(res);
      })
      .finally(() => {
        loading.value = false;
      });
  }

  defineExpose({ openModal });
</script>

<style lang="less" scoped>
  .detail-section {
    margin-top: 16px;
  }

  .detail-section-title {
    margin-bottom: 8px;
    font-size: 14px;
    font-weight: 600;
  }

  .attendee-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 10px;
  }

  .attendee-tag {
    margin: 0;
  }

  .recording-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 0;
    border-bottom: 1px solid #f0f0f0;
  }

  .recording-item:last-child {
    border-bottom: none;
  }

  .recording-name {
    font-weight: 500;
  }

  .recording-link {
    color: #2080f0;
    cursor: pointer;
  }

  .recording-status {
    color: #999;
  }

  .detail-empty {
    color: #999;
    font-size: 13px;
  }
</style>
