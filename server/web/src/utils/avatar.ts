import defaultAvatar from '@/assets/images/default-avatar.png';

/** HotGo 历史默认考拉头像 */
const LEGACY_DEFAULT_AVATAR_MARKERS = [
  'cqdq8er9nfkchdopav.png',
  'gmycos.facms.cn/hotgo/attachment/2023-02-09/cqdq8er9',
];

/** 本地默认头像路径（库内 / 配置） */
const LOCAL_DEFAULT_AVATAR_PATHS = ['/default-avatar.png', '/resource/image/default-avatar.png'];

export function isDefaultOrLegacyAvatar(src?: string | null): boolean {
  if (!src) return true;
  if (LOCAL_DEFAULT_AVATAR_PATHS.some((p) => src === p || src.endsWith(p))) return true;
  return LEGACY_DEFAULT_AVATAR_MARKERS.some((m) => src.includes(m));
}

/** 展示用头像：空、旧默认、本地默认路径 → 统一用内置默认图 */
export function resolveAvatar(src?: string | null): string {
  if (isDefaultOrLegacyAvatar(src)) {
    return defaultAvatar;
  }
  return src as string;
}

export { defaultAvatar };
