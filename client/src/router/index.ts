import { createRouter, createWebHistory } from 'vue-router'
import { isLoggedIn } from '@/stores/auth'
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

export default router
