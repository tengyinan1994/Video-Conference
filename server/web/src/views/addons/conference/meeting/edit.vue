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
                  <n-input v-model:value="formValue.title" placeholder="请输入会议名称" maxlength="64" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="主持人" path="hostName">
                  <n-input v-model:value="formValue.hostName" placeholder="默认当前管理员" />
                </n-form-item>
              </n-gi>
              <n-gi>
                <n-form-item label="会议时间" path="startAt">
                  <DatePicker
                    v-model:startValue="formValue.startAt"
                    v-model:endValue="formValue.endAt"
                    type="datetimerange"
                  />
                </n-form-item>
              </n-gi>
              <n-gi v-if="formValue.id > 0">
                <n-form-item label="状态">
                  <n-tag :type="statusTagType" size="small">
                    {{ dict.getLabel('MeetingStatusOptions', formValue.status) || formValue.status }}
                  </n-tag>
                </n-form-item>
              </n-gi>
              <n-gi v-if="formValue.id > 0">
                <n-form-item label="房间名">
                  <n-input :value="formValue.roomName" disabled />
                </n-form-item>
              </n-gi>
              <n-gi v-if="formValue.id > 0">
                <n-form-item label="分享码">
                  <n-input :value="formValue.shareCode" disabled />
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
  import { ref, computed } from 'vue';
  import { Edit, View } from '@/api/addons/conference/meeting';
  import { State, newState, rules } from './model';
  import { useProjectSettingStore } from '@/store/modules/projectSetting';
  import { useMessage } from 'naive-ui';
  import { adaModalWidth } from '@/utils/hotgo';
  import { useDictStore } from '@/store/modules/dict';
  import DatePicker from '@/components/DatePicker/datePicker.vue';

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
    if (!state || state.id < 1) {
      formValue.value = newState(null);
      return;
    }
    loading.value = true;
    View({ id: state.id })
      .then((res) => {
        formValue.value = newState(res);
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

<style lang="less"></style>
