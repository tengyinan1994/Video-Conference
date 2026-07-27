import { createRouter, createWebHistory } from 'vue-router'
import JoinView from '@/views/JoinView.vue'
import RoomView from '@/views/RoomView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'join', component: JoinView },
    { path: '/room/:room', name: 'room', component: RoomView },
  ],
})

export default router
