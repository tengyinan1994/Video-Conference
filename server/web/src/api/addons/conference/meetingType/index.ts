import { http } from '@/utils/http/axios';

// 获取会议类型列表
export function List(params) {
  return http.request({
    url: '/conference/meetingType/list',
    method: 'get',
    params,
  });
}

// 获取会议类型详情
export function View(params) {
  return http.request({
    url: '/conference/meetingType/view',
    method: 'GET',
    params,
  });
}

// 新增/编辑会议类型
export function Edit(params) {
  return http.request({
    url: '/conference/meetingType/edit',
    method: 'POST',
    params,
  });
}

// 删除/批量删除会议类型
export function Delete(params) {
  return http.request({
    url: '/conference/meetingType/delete',
    method: 'POST',
    params,
  });
}

// 会议类型选项（不分页，供会议编辑下拉使用）
export function Option() {
  return http.request({
    url: '/conference/meetingType/option',
    method: 'GET',
  });
}
