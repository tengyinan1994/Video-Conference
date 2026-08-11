import { http } from '@/utils/http/axios';

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
