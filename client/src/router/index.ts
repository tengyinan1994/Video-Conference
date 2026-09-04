import { createRouter, createWebHistory } from 'vue-router'
import { isLoggedIn, subscribeAuth } from '@/stores/auth'
import JoinView from '@/views/JoinView.vue'
import LobbyView from '@/views/LobbyView.vue'
import LoginView from '@/views/LoginView.vue'
import RoomView from '@/views/RoomView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    { path: '/', name: 'lobby', component: LobbyView, meta: { requiresAuth: true } },
    {
      path: '/join/:shareCode',
      name: 'join',
      component: JoinView,
      meta: { public: true },
    },
    { path: '/room/:room', name: 'room', component: RoomView },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach((to) => {
  const loggedIn = isLoggedIn()
  if (to.meta.requiresAuth && !loggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && loggedIn) {
    const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : ''
    if (redirect.startsWith('/')) return redirect
    return { name: 'lobby' }
  }
  return true
})

// 登录态被清除时（401/61，例如后台删除或禁用了当前账号），
// 主动跳回登录页，避免停留在受保护路由上出现空白/无限转圈。
// 注：清除登录态本身不会触发导航，只有在这里监听变化才能及时回到登录页。
let redirectingToLogin = false

subscribeAuth(() => {
  if (isLoggedIn() || redirectingToLogin) return
  const route = router.currentRoute.value
  const needsAuth = route.matched.some((r) => r.meta.requiresAuth)
  if (!needsAuth) return
  redirectingToLogin = true
  void router
    .replace({ name: 'login', query: { redirect: route.fullPath } })
    .finally(() => {
      redirectingToLogin = false
    })
})

export default router
