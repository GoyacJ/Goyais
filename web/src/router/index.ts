import { createRouter, createWebHistory } from 'vue-router'
import AssetsView from '@/views/AssetsView.vue'
import CanvasView from '@/views/CanvasView.vue'
import CommandsView from '@/views/CommandsView.vue'
import HomeView from '@/views/HomeView.vue'
import PluginsView from '@/views/PluginsView.vue'
import SettingsView from '@/views/SettingsView.vue'
import StreamsView from '@/views/StreamsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/canvas', name: 'canvas', component: CanvasView },
    { path: '/commands', name: 'commands', component: CommandsView },
    { path: '/assets', name: 'assets', component: AssetsView },
    { path: '/plugins', name: 'plugins', component: PluginsView },
    { path: '/streams', name: 'streams', component: StreamsView },
    { path: '/settings', name: 'settings', component: SettingsView },
  ],
})

export default router
