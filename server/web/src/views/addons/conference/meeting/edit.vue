<template>
  <div>
    <n-modal
      v-model:show="showModal"
      :mask-closable="false"
      :show-icon="false"
      preset="dialog"
      transform-origin="center"
      :title="formValue.id > 0 ? '编辑会议 #' + formValue.id : '新建会议'"
      :style="{ width: dialogWidth }"
    >
      <n-scrollbar style="max-height: 87vh" class="pr-5">
        <n-spin :show="loading" description="请稍候...">
          <n-form
            ref="formRef"
            :model="formValue"
            :rules="rules"
            :label-placement="settingStore.isMobile ? 'top' : 'left'"
            :label-width="100"
            class="py-4"
          >
            <n-grid cols="1" responsive="screen">
              <n-gi>
                <n-form-item label="会议名称" path="title">
                  <n-input
                    v-model:value="formValue.title"
                    placeholder="请输入会议名称"
                    maxlength="64"
                  />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="主持人" path="hostName">
                  <n-input v-model:value="formValue.hostName" placeholder="默认当前管理员" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="会议类型" path="typeId">
                  <n-select
                    v-model:value="typeIdValue"
                    :options="typeOptions"
                    :disabled="typeOptions.length < 1"
                    clearable
                    placeholder="非必选，不选择则不计类型"
                  />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="开始时间" path="startAt">
                  <div class="meeting-time-row">
                    <n-date-picker
                      v-model:value="startDateTs"
                      type="date"
                      :clearable="true"
                      :shortcuts="dateShortcuts"
                      :is-date-disabled="isStartDateDisabled"
                      placeholder="选择开始日期"
                      class="meeting-time-date"
                    />
                    <n-time-picker
                      v-model:value="startTimeTs"
                      format="HH:mm"
                      clearable
                      placeholder="选择开始时间"
                      class="meeting-time-time"
                    />
                  </div>
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="结束时间" path="endAt">
                  <div class="meeting-time-row">
                    <n-date-picker
                      v-model:value="endDateTs"
                      type="date"
                      :clearable="true"
                      :shortcuts="dateShortcuts"
                      placeholder="选择结束日期"
                      :is-date-disabled="isEndDateDisabled"
                      class="meeting-time-date"
                    />
                    <n-time-picker
                      v-model:value="endTimeTs"
                      format="HH:mm"
                      clearable
                      placeholder="选择结束时间"
                      class="meeting-time-time"
                    />
                  </div>
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="开启录制" path="recordEnabled">
                  <n-switch v-model:value="formValue.recordEnabled" />
                  <span class="ml-3 text-gray-500 text-sm">默认关；开启后首位参会者进房自动录</span>
                </n-form-item>
              </n-gi>
              <n-gi v-if="formValue.id > 0">
                <n-form-item label="状态">
                  <n-tag :type="statusTagType" size="small">
                    {{
                      dict.getLabel('MeetingStatusOptions', formValue.status) || formValue.status
                    }}
                  </n-tag>
                </n-form-item>
              </n-gi>
            </n-grid>
          </n-form>
        </n-spin>
      </n-scrollbar>
      <template #action>
        <n-space>
          <n-button @click="closeForm">取消</n-button>
          <n-button type="info" :loading="formBtnLoading" @click="confirmForm">确定</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script lang="ts" setup>
  import { ref, computed, watch } from 'vue';
  import { format } from 'date-fns';
  import { Edit, View } from '@/api/addons/conference/meeting';
  import { Option } from '@/api/addons/conference/meetingType';
  import { State, newState, rules } from './model';
  import { useProjectSettingStore } from '@/store/modules/projectSetting';
  import { useMessage } from 'naive-ui';
  import { adaModalWidth } from '@/utils/hotgo';
  import { useDictStore } from '@/store/modules/dict';
  import { defShortcuts } from '@/utils/dateUtil';

  const emit = defineEmits(['reloadTable']);
  const dict = useDictStore();
  const message = useMessage();
  const settingStore = useProjectSettingStore();
  const loading = ref(false);
  const showModal = ref(false);
  const formValue = ref<State>(newState(null));
  const formRef = ref<any>({});
  const formBtnLoading = ref(false);
  const dialogWidth = computed(() => adaModalWidth(640));

  // 会议类型选项：0（未分类）用 null 表示，清空时回写 0
  const typeOptions = ref<Array<{ label: string; value: number }>>([]);
  const typeIdValue = computed({
    get: () => (formValue.value.typeId > 0 ? formValue.value.typeId : null),
    set: (value: number | null) => {
      formValue.value.typeId = value ?? 0;
    },
  });

  function loadTypeOptions() {
    Option()
      .then((res) => {
        const list = res?.list || [];
        typeOptions.value = list.map((item) => ({ label: item.name, value: item.id }));
      })
      .catch(() => {
        typeOptions.value = [];
      });
  }

  const DATETIME_FMT = 'yyyy-MM-dd HH:mm:ss';
  const dateShortcuts = defShortcuts();
  const startDateTs = ref<number | null>(null);
  const startTimeTs = ref<number | null>(null);
  const endDateTs = ref<number | null>(null);
  const endTimeTs = ref<number | null>(null);

  // 把完整的 datetime 拆成「日期」与「时间」两个独立可编辑的量
  function splitDateTime(value: string | number | null) {
    if (value === null || value === undefined || value === '') {
      return { date: null, time: null };
    }
    const d = new Date(value);
    if (isNaN(d.getTime())) {
      return { date: null, time: null };
    }
    const date = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
    const time = new Date(1970, 0, 1, d.getHours(), d.getMinutes(), d.getSeconds()).getTime();
    return { date, time };
  }

  // 由「日期 + 时间」组合回完整的 datetime 字符串
  function composeDateTime(dateTs: number | null, timeTs: number | null): string | null {
    if (dateTs === null || timeTs === null) {
      return null;
    }
    const d = new Date(dateTs);
    const t = new Date(timeTs);
    d.setHours(t.getHours(), t.getMinutes(), t.getSeconds(), 0);
    return format(d, DATETIME_FMT);
  }

  function isEndDateDisabled(timestamp: number) {
    if (startDateTs.value === null) {
      return false;
    }
    const start = new Date(startDateTs.value);
    const startDay = new Date(start.getFullYear(), start.getMonth(), start.getDate()).getTime();
    return timestamp < startDay;
  }

  // 新建会议时，开始日期不能早于今天；编辑历史会议（可能进行中/已结束）不限制
  function isStartDateDisabled(timestamp: number) {
    if (formValue.value.id > 0) {
      return false;
    }
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    return timestamp < today;
  }

  function initTimeSplit() {
    const s = splitDateTime(formValue.value.startAt);
    const e = splitDateTime(formValue.value.endAt);
    startDateTs.value = s.date;
    startTimeTs.value = s.time;
    endDateTs.value = e.date;
    endTimeTs.value = e.time;
  }

  watch([startDateTs, startTimeTs], () => {
    formValue.value.startAt = composeDateTime(startDateTs.value, startTimeTs.value);
  });
  watch([endDateTs, endTimeTs], () => {
    formValue.value.endAt = composeDateTime(endDateTs.value, endTimeTs.value);
  });

  const statusTagType = computed(() => {
    const map: Record<string, string> = {
      scheduled: 'info',
      ongoing: 'success',
      ended: 'default',
    };
    return map[formValue.value.status] || 'default';
  });

  function openModal(state: State | null) {
    showModal.value = true;
    loadTypeOptions();
    if (!state || state.id < 1) {
      formValue.value = newState(null);
      initTimeSplit();
      return;
    }
    loading.value = true;
    View({ id: state.id })
      .then((res) => {
        formValue.value = newState(res);
        initTimeSplit();
      })
      .finally(() => {
        loading.value = false;
      });
  }

  function confirmForm(e) {
    e.preventDefault();
    formBtnLoading.value = true;
    formRef.value.validate((errors) => {
      if (!errors) {
        if (!formValue.value.startAt || !formValue.value.endAt) {
          message.error('请填写会议时间');
          formBtnLoading.value = false;
          return;
        }
        if (
          new Date(formValue.value.startAt).getTime() >= new Date(formValue.value.endAt).getTime()
        ) {
          message.error('结束时间必须晚于开始时间');
          formBtnLoading.value = false;
          return;
        }
        if (formValue.value.id <= 0 && new Date(formValue.value.startAt).getTime() < Date.now()) {
          message.error('开始时间不能早于当前时间');
          formBtnLoading.value = false;
          return;
        }
        Edit(formValue.value)
          .then(() => {
            message.success('操作成功');
            closeForm();
            emit('reloadTable');
          })
          .finally(() => {
            formBtnLoading.value = false;
          });
      } else {
        message.error('请填写完整信息');
        formBtnLoading.value = false;
      }
    });
  }

  function closeForm() {
    showModal.value = false;
    loading.value = false;
  }

  defineExpose({ openModal });
</script>

<style lang="less" scoped>
  .meeting-time-row {
    display: flex;
    width: 100%;
    gap: 12px;
  }

  .meeting-time-date,
  .meeting-time-time {
    flex: 1 1 0;
    min-width: 0;
  }
</style>
