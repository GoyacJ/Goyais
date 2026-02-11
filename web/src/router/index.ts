import { createRouter, createWebHistory } from 'vue-router'
import AssetsView from '@/views/AssetsView.vue'
import AIWorkbenchView from '@/views/AIWorkbenchView.vue'
import CanvasView from '@/views/CanvasView.vue'
import CommandsView from '@/views/CommandsView.vue'
import ForbiddenView from '@/views/ForbiddenView.vue'
import HomeView from '@/views/HomeView.vue'
import NotFoundView from '@/views/NotFoundView.vue'
import PluginsView from '@/views/PluginsView.vue'
import SettingsView from '@/views/SettingsView.vue'
import StreamsView from '@/views/StreamsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
      meta: { layoutDefault: 'topnav', windowed: true, windowManifestKey: 'home' },
    },
    {
      path: '/canvas',
      name: 'canvas',
      component: CanvasView,
      meta: { layoutDefault: 'focus', windowed: true, windowManifestKey: 'canvas' },
    },
    {
      path: '/ai',
      name: 'ai-workbench',
      component: AIWorkbenchView,
      meta: { layoutDefault: 'console', windowed: true, windowManifestKey: 'ai-workbench' },
    },
    {
      path: '/commands',
      name: 'commands',
      component: CommandsView,
      meta: { layoutDefault: 'console', windowed: true, windowManifestKey: 'commands' },
    },
    {
      path: '/assets',
      name: 'assets',
      component: AssetsView,
      meta: { layoutDefault: 'console', windowed: true, windowManifestKey: 'assets' },
    },
    {
      path: '/plugins',
      name: 'plugins',
      component: PluginsView,
      meta: { layoutDefault: 'topnav', windowed: true, windowManifestKey: 'plugins' },
    },
    {
      path: '/streams',
      name: 'streams',
      component: StreamsView,
      meta: { layoutDefault: 'topnav', windowed: true, windowManifestKey: 'streams' },
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsView,
      meta: { layoutDefault: 'topnav', windowed: true, windowManifestKey: 'settings' },
    },
    {
      path: '/forbidden',
      name: 'forbidden',
      component: ForbiddenView,
      meta: { layoutDefault: 'topnav', windowed: true, windowManifestKey: 'forbidden' },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFoundView,
      meta: { layoutDefault: 'topnav', windowed: true, windowManifestKey: 'not-found' },
    },
  ],
})

export default router
