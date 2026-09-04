import type { RouteRecordRaw } from 'vue-router';
import { isNavigationFailure, Router } from 'vue-router';
import { useUserStoreWidthOut } from '@/store/modules/user';
import { useAsyncRouteStoreWidthOut } from '@/store/modules/asyncRoute';
import { ACCESS_TOKEN, CURRENT_USER } from '@/store/mutation-types';
import { storage } from '@/utils/Storage';
import { PageEnum } from '@/enums/pageEnum';
import { ErrorPageRoute } from '@/router/base';
import { jump } from '@/utils/http/axios';
import { getNowUrl } from '@/utils/urlUtils';

const LOGIN_PATH = PageEnum.BASE_LOGIN;
const whitePathList = [LOGIN_PATH]; // no redirect whitelist

/** 路由树中的节点结构（generateRoutes 产出的任意形态） */
interface MenuRouteLike {
  path?: string;
  component?: unknown;
  children?: MenuRouteLike[];
}

/** 收集路由树中所有路径（含目录与叶子），用于判断目标页面是否在当前角色可访问菜单内 */
function collectRoutePaths(routes: MenuRouteLike[], out: Set<string>): void {
  (routes || []).forEach((r) => {
    if (r.path) out.add(r.path);
    if (r.children && r.children.length) {
      collectRoutePaths(r.children, out);
    }
  });
}

/** 取第一个可访问的菜单页面路径（叶子节点，如 /conference/meeting） */
function firstMenuPath(routes: MenuRouteLike[]): string {
  for (const r of routes || []) {
    if (r.children && r.children.length) {
      const p = firstMenuPath(r.children);
      if (p) return p;
    } else if (r.component && r.path) {
      return r.path;
    }
  }
  return '';
}

export function createRouterGuards(router: Router) {
  const userStore = useUserStoreWidthOut();
  const asyncRouteStore = useAsyncRouteStoreWidthOut();
  router.beforeEach(async (to, from, next) => {
    const Loading = window['$loading'] || null;
    Loading && Loading.start();

    if (from.path === LOGIN_PATH && to.name === 'errorPage') {
      next(PageEnum.BASE_HOME);
      return;
    }

    // Whitelist can be directly entered
    if (whitePathList.includes(to.path as PageEnum)) {
      await userStore.LoadLoginConfig();
      next();
      return;
    }

    const token = storage.get(ACCESS_TOKEN);

    if (!token) {
      // You can access without permissions. You need to set the routing meta.ignoreAuth to true
      if (to.meta.ignoreAuth) {
        next();
        return;
      }

      // redirect login page
      const redirectData: { path: string; replace: boolean; query?: Recordable<string> } = {
        path: LOGIN_PATH,
        replace: true,
      };
      if (to.path) {
        redirectData.query = {
          ...redirectData.query,
          redirect: to.path,
        };
      }
      next(redirectData);
      return;
    }

    if (asyncRouteStore.getIsDynamicAddedRoute) {
      next();
      return;
    }

    const redirectPath = (from.query.redirect || to.path) as string;
    const redirect = decodeURIComponent(redirectPath);
    const nextData = to.path === redirect ? { ...to, replace: true } : { path: redirect };

    // 获取登录用户信息。若账号已在后台被删除/禁用或登录身份失效，后端会返回鉴权错误；
    // 此时必须清空本地登录态并跳回登录页，避免页面停留在空白加载状态（转圈）。
    let userInfo;
    try {
      userInfo = await userStore.GetInfo();
      await userStore.LoadLoginConfig();
    } catch (error) {
      console.error('获取登录用户信息失败，跳转登录页', error);
      storage.remove(ACCESS_TOKEN);
      storage.remove(CURRENT_USER);
      userStore.setToken('');
      userStore.setUserInfo(null);
      next({
        path: LOGIN_PATH,
        replace: true,
        query: { redirect: to.path },
      });
      Loading && Loading.finish();
      return;
    }

    // 是否允许获取微信openid
    if (userStore.allowWxOpenId()) {
      let path = nextData.path;
      if (path === LOGIN_PATH) {
        path = PageEnum.BASE_HOME_REDIRECT;
      }

      const URI = getNowUrl() + '#' + path;
      jump('/wechat/authorize', { type: 'openId', syncRedirect: URI });
      return;
    }

    await userStore.GetConfig();
    const routes = await asyncRouteStore.generateRoutes(userInfo);

    // 动态添加可访问路由表
    routes.forEach((item) => {
      router.addRoute(item as unknown as RouteRecordRaw);
    });

    //添加404
    const isErrorPage = router.getRoutes().findIndex((item) => item.name === ErrorPageRoute.name);
    if (isErrorPage === -1) {
      router.addRoute(ErrorPageRoute as unknown as RouteRecordRaw);
    }

    asyncRouteStore.setDynamicAddedRoute(true);

    // 登录后的落点规避权限404：新角色(如普通员工)可能没有默认首页菜单权限，
    // 或浏览器残留的 redirect 指向了无权限页面(如 /org/user)。
    // 若目标页面不在其可访问菜单中，优先回退到默认首页(会议列表)；默认首页也不可访问时再取第一个可访问菜单。
    const accessiblePaths = new Set<string>();
    collectRoutePaths(routes as MenuRouteLike[], accessiblePaths);
    let safePath = (nextData as { path?: string }).path || '';
    if (safePath === '/' || safePath === LOGIN_PATH) {
      safePath = PageEnum.BASE_HOME;
    }
    if (!accessiblePaths.has(safePath)) {
      if (accessiblePaths.has(PageEnum.BASE_HOME)) {
        safePath = PageEnum.BASE_HOME;
      } else {
        const first = firstMenuPath(routes as MenuRouteLike[]);
        if (first) {
          safePath = first;
        }
      }
    }
    if (safePath === (nextData as { path?: string }).path) {
      next(nextData);
    } else {
      next({ path: safePath, replace: true });
    }
    Loading && Loading.finish();
  });

  router.afterEach((to, _, failure) => {
    document.title = (to?.meta?.title as string) || document.title;
    if (isNavigationFailure(failure)) {
      //console.log('failed navigation', failure)
    }
    const Loading = window['$loading'] || null;
    Loading && Loading.finish();
  });

  router.onError((error) => {
    console.log(error, '路由错误');
  });
}
