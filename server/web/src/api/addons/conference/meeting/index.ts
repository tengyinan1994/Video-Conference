import { http } from '@/utils/http/axios';
import { useGlobSetting } from '@/hooks/setting';
import { useUserStoreWidthOut } from '@/store/modules/user';
import { encodeParams } from '@/utils/urlUtils';

/** 后台回放/下载走同源 HTTPS 代理（<a> 无法带 Authorization 头，token 放 query） */
export function recordingProxyUrl(kind: 'play' | 'download', id: number) {
  const prefix = useGlobSetting().urlPrefix || '';
  const token = useUserStoreWidthOut().token || '';
  return `${prefix}/conference/recording/${kind}?${encodeParams({ id, authorization: token })}`;
}

// 获取会议列表
export function List(params) {
  return http.request({
    url: '/conference/meeting/list',
    method: 'get',
    params,
  });
}

// 删除/批量删除会议
export function Delete(params) {
  return http.request({
    url: '/conference/meeting/delete',
    method: 'POST',
    params,
  });
}

// 新增/编辑会议
export function Edit(params) {
  return http.request({
    url: '/conference/meeting/edit',
    method: 'POST',
    params,
  });
}

// 获取会议详情
export function View(params) {
  return http.request({
    url: '/conference/meeting/view',
    method: 'GET',
    params,
  });
}

// 结束会议
export function Release(params) {
  return http.request({
    url: '/conference/meeting/release',
    method: 'POST',
    params,
  });
}
