<template>
  <div>
    <div class="n-layout-page-header">
      <n-card :bordered="false" title="会议管理">
        管理员可查看全部会议，并对任意状态的会议进行编辑、结束或删除。
      </n-card>
    </div>
    <n-card :bordered="false" class="proCard">
      <BasicForm
        ref="searchFormRef"
        @register="register"
        @submit="reloadTable"
        @reset="reloadTable"
        @keyup.enter="reloadTable"
      />
      <BasicTable
        ref="actionRef"
        openChecked
        :columns="columns"
        :request="loadDataTable"
        :row-key="(row) => row.id"
        :actionColumn="actionColumn"
        :scroll-x="scrollX"
        :resizeHeightOffset="-10000"
        :checked-row-keys="checkedIds"
        @update:checked-row-keys="handleOnCheckedRow"
        @fetch-success="handleFetchSuccess"
      >
        <template #tableTitle>
          <n-button
            type="primary"
            class="min-left-space"
            v-if="hasPermission(['/conference/meeting/edit'])"
            @click="addTable"
          >
            <template #icon>
              <n-icon>
                <PlusOutlined />
              </n-icon>
            </template>
            新建会议
          </n-button>
          <n-button
            type="error"
            class="min-left-space"
            v-if="hasPermission(['/conference/meeting/delete'])"
            @click="handleBatchDelete"
          >
            <template #icon>
              <n-icon>
                <DeleteOutlined />
              </n-icon>
            </template>
            批量删除
          </n-button>
        </template>
      </BasicTable>
    </n-card>
    <Edit ref="editRef" @reloadTable="reloadTable" />
  </div>
</template>

<script lang="ts" setup>
  import { h, reactive, ref, computed, onMounted } from 'vue';
  import { useDialog, useMessage } from 'naive-ui';
  import { BasicTable, TableAction } from '@/components/Table';
  import { BasicForm, useForm } from '@/components/Form/index';
  import { usePermission } from '@/hooks/web/usePermission';
  import { List, Delete, Release } from '@/api/addons/conference/meeting';
  import { PlusOutlined, DeleteOutlined } from '@vicons/antd';
  import { columns, schemas, loadOptions } from './model';
  import { adaTableScrollX } from '@/utils/hotgo';
  import Edit from './edit.vue';

  const dialog = useDialog();
  const message = useMessage();
  const { hasPermission } = usePermission();
  const actionRef = ref();
  const searchFormRef = ref<any>({});
  const editRef = ref();
  const checkedIds = ref([]);

  const actionColumn = reactive({
    width: 220,
    title: '操作',
    key: 'action',
    fixed: 'right',
    render(record) {
      return h(TableAction as any, {
        style: 'button',
        actions: [
          {
            label: '编辑',
            onClick: handleEdit.bind(null, record),
            auth: ['/conference/meeting/edit'],
          },
          {
            label: '结束',
            onClick: handleRelease.bind(null, record),
            ifShow: () => record.status !== 'ended',
            auth: ['/conference/meeting/release'],
          },
          {
            label: '删除',
            onClick: handleDelete.bind(null, record),
            auth: ['/conference/meeting/delete'],
          },
        ],
      });
    },
  });

  const scrollX = computed(() => {
    return adaTableScrollX(columns, actionColumn.width);
  });

  const [register, {}] = useForm({
    gridProps: { cols: '1 s:1 m:2 l:3 xl:4 2xl:4' },
    labelWidth: 80,
    schemas,
  });

  const loadDataTable = async (res) => {
    return await List({ ...searchFormRef.value?.formModel, ...res });
  };

  function handleOnCheckedRow(rowKeys) {
    checkedIds.value = rowKeys;
  }

  // 数据刷新/翻页/搜索后重置勾选，避免残留其它页的 id 导致批量删除数量不准、误删
  function handleFetchSuccess() {
    checkedIds.value = [];
  }

  function reloadTable() {
    actionRef.value?.reload();
  }

  function addTable() {
    editRef.value.openModal(null);
  }

  function handleEdit(record: Recordable) {
    editRef.value.openModal(record);
  }

  function handleDelete(record: Recordable) {
    dialog.warning({
      title: '警告',
      content: `确定删除会议「${record.title}」？删除后不可恢复。`,
      positiveText: '确定',
      negativeText: '取消',
      onPositiveClick: () => {
        Delete(record).then(() => {
          message.success('删除成功');
          reloadTable();
        });
      },
    });
  }

  function handleBatchDelete() {
    // 兜底：仅删除当前表格可见行的 id，避免残留的其它页勾选被一并删除
    const rows = actionRef.value?.getDataSource?.() || [];
    const pageIds = new Set(rows.map((row) => row.id));
    const ids = checkedIds.value.filter((id) => pageIds.has(id));
    if (ids.length < 1) {
      checkedIds.value = [];
      message.error('请至少选择一项要删除的数据');
      return;
    }
    dialog.warning({
      title: '警告',
      content: `确定批量删除选中的 ${ids.length} 场会议？删除后不可恢复。`,
      positiveText: '确定',
      negativeText: '取消',
      onPositiveClick: () => {
        Delete({ id: ids }).then(() => {
          checkedIds.value = [];
          message.success('删除成功');
          reloadTable();
        });
      },
    });
  }

  function handleRelease(record: Recordable) {
    dialog.warning({
      title: '结束会议',
      content: `确定结束会议「${record.title}」？结束后将保留历史记录。`,
      positiveText: '确定',
      negativeText: '取消',
      onPositiveClick: () => {
        Release({ id: record.id }).then(() => {
          message.success('会议已结束');
          reloadTable();
        });
      },
    });
  }

  onMounted(() => {
    loadOptions();
  });
</script>

<style lang="less" scoped></style>
