import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: () => import('@/views/HomeView.vue'), meta: { title: '今日我最美' } },
  {
    path: '/create',
    name: 'create',
    component: () => import('@/views/CreateRoomView.vue'),
    meta: { title: '開一間房 - 今日我最美' },
  },
  {
    path: '/join/:roomId?',
    name: 'join',
    component: () => import('@/views/JoinRoomView.vue'),
    props: true,
    meta: { title: '加入遊戲 - 今日我最美' },
  },
  {
    path: '/lobby/:roomId',
    name: 'lobby',
    component: () => import('@/views/LobbyView.vue'),
    props: true,
    meta: { title: '等待大廳 - 今日我最美' },
  },
  {
    path: '/game/host/:roomId',
    name: 'game-host',
    component: () => import('@/views/GameHostView.vue'),
    props: true,
    meta: { title: '主畫面 - 今日我最美' },
  },
  {
    path: '/game/player/:roomId',
    name: 'game-player',
    component: () => import('@/views/GamePlayerView.vue'),
    props: true,
    meta: { title: '拍照中 - 今日我最美' },
  },
  {
    path: '/results/:roomId',
    name: 'results',
    component: () => import('@/views/ResultsView.vue'),
    props: true,
    meta: { title: '結算 - 今日我最美' },
  },
  {
    path: '/admin',
    name: 'admin',
    component: () => import('@/views/AdminView.vue'),
    meta: { title: '題目示範圖後台 - 今日我最美' },
  },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach((to, _from, next) => {
  if (to.meta?.title) document.title = to.meta.title as string
  next()
})

export default router
