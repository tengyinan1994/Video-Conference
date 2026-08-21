import { h, ref } from 'vue';
import { cloneDeep } from 'lodash-es';
import { FormSchema } from '@/components/Form';
import { defRangeShortcuts } from '@/utils/dateUtil';
import { renderOptionTag } from '@/utils';
import { useDictStore } from '@/store/modules/dict';
import type { FormRules } from 'naive-ui/es/form/src/interface';

const dict = useDictStore();

export class State {
  public id = 0;
  public title = '';
  public roomName = '';
  public hostId = 0;
  public hostName = '';
  public startAt: string | number | null = null;
  public endAt: string | number | null = null;
  public status = '';
  public shareCode = '';
  public shareUrl = '';
  public tab = '';
  public createdBy = 0;
  public createdAt = '';
  public updatedAt = '';
  public releasedAt = '';
  public recordEnabled = false;
  public recordings: Array<{
    id: number;
    seq: number;
    status: string;
    playUrl?: string;
    downloadUrl?: string;
  }> = [];

  constructor(state?: Partial<State>) {
    if (state) {
      Object.assign(this, state);
    }
  }
}

export function newState(state: State | Record<string, any> | null): State {
  if (state !== null) {
    if (state instanceof State) {
      return cloneDeep(state);
    }
    return new State(state);
  }
  return new State();
}

export const rules: FormRules = {
  title: {
    required: true,
    trigger: ['blur', 'input'],
    type: 'string',
    message: '请输入会议名称',
  },
  startAt: {
    required: true,
    trigger: ['blur', 'change'],
    type: 'any',
    message: '请选择开始时间',
  },
  endAt: {
    required: true,
    trigger: ['blur', 'change'],
    type: 'any',
    message: '请选择结束时间',
  },
};

export const schemas = ref<FormSchema[]>([
  {
    field: 'keyword',
    component: 'NInput',
    label: '关键词',
    componentProps: {
      placeholder: '名称/主持人/房间/分享码',
    },
  },
  {
    field: 'title',
    component: 'NInput',
    label: '会议名称',
    componentProps: {
      placeholder: '请输入会议名称',
    },
  },
  {
    field: 'hostName',
    component: 'NInput',
    label: '主持人',
    componentProps: {
      placeholder: '请输入主持人',
    },
  },
  {
    field: 'status',
    component: 'NSelect',
    label: '状态',
    defaultValue: null,
    componentProps: {
      clearable: true,
      placeholder: '请选择状态',
      options: dict.getOption('MeetingStatusOptions'),
    },
  },
  {
    field: 'startAt',
    component: 'NDatePicker',
    label: '开始时间',
    componentProps: {
      type: 'datetimerange',
      clearable: true,
      shortcuts: defRangeShortcuts(),
    },
  },
]);

export const columns = [
  {
    title: 'ID',
    key: 'id',
    width: 80,
  },
  {
    title: '会议名称',
    key: 'title',
    width: 180,
  },
  {
    title: '主持人',
    key: 'hostName',
    width: 120,
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render(row) {
      return renderOptionTag('MeetingStatusOptions', row.status);
    },
  },
  {
    title: '开始时间',
    key: 'startAt',
    width: 180,
  },
  {
    title: '结束时间',
    key: 'endAt',
    width: 180,
  },
  {
    title: '房间名',
    key: 'roomName',
    width: 160,
  },
  {
    title: '分享码',
    key: 'shareCode',
    width: 140,
  },
  {
    title: '录制',
    key: 'recordings',
    width: 240,
    render(row) {
      const segs = Array.isArray(row.recordings) ? row.recordings : [];
      if (!segs.length) {
        return row.recordEnabled ? '已开启（暂无文件）' : '未开启';
      }
      return h(
        'div',
        { style: 'display:flex;flex-wrap:wrap;gap:6px 10px;' },
        segs.map((seg) => {
          if (seg.playUrl || seg.downloadUrl) {
            const play = seg.playUrl
              ? h(
                  'a',
                  {
                    href: seg.playUrl,
                    target: '_blank',
                    rel: 'noopener',
                    style: 'margin-right:8px',
                  },
                  `第${seg.seq}段回放`
                )
              : null;
            const download = h(
              'a',
              {
                href: seg.downloadUrl || seg.playUrl,
                download: `recording-${row.id}-${seg.seq}.mp4`,
                rel: 'noopener',
              },
              play ? '下载' : `第${seg.seq}段下载`
            );
            return h('span', { style: 'margin-right:10px' }, [play, download].filter(Boolean));
          }
          const label =
            seg.status === 'failed'
              ? `第${seg.seq}段(失败)`
              : seg.status === 'complete'
                ? `第${seg.seq}段(无文件)`
                : `第${seg.seq}段(${seg.status || '处理中'})`;
          return h('span', label);
        })
      );
    },
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 180,
  },
];

export function loadOptions() {
  dict.loadOptions(['MeetingStatusOptions']);
}
