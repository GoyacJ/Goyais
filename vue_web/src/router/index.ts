/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { createRouter, createWebHistory } from 'vue-router'
import AssetsView from '@/views/AssetsView.vue'
import AIWorkbenchView from '@/views/AIWorkbenchView.vue'
import AlgorithmLibraryView from '@/views/AlgorithmLibraryView.vue'
import CanvasView from '@/views/CanvasView.vue'
import CommandsView from '@/views/CommandsView.vue'
import ContextBundleView from '@/views/ContextBundleView.vue'
import ForbiddenView from '@/views/ForbiddenView.vue'
import HomeView from '@/views/HomeView.vue'
import NotFoundView from '@/views/NotFoundView.vue'
import PermissionManagementView from '@/views/PermissionManagementView.vue'
import PluginsView from '@/views/PluginsView.vue'
import RunCenterView from '@/views/RunCenterView.vue'
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
      path: '/run-center',
      name: 'run-center',
      component: RunCenterView,
      meta: { layoutDefault: 'console', windowed: true, windowManifestKey: 'run-center' },
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
      path: '/algorithm-library',
      name: 'algorithm-library',
      component: AlgorithmLibraryView,
      meta: { layoutDefault: 'console', windowed: true, windowManifestKey: 'algorithm-library' },
    },
    {
      path: '/streams',
      name: 'streams',
      component: StreamsView,
      meta: { layoutDefault: 'topnav', windowed: true, windowManifestKey: 'streams' },
    },
    {
      path: '/permissions',
      name: 'permission-management',
      component: PermissionManagementView,
      meta: { layoutDefault: 'console', windowed: true, windowManifestKey: 'permission-management' },
    },
    {
      path: '/context-bundles',
      name: 'context-bundles',
      component: ContextBundleView,
      meta: { layoutDefault: 'console', windowed: true, windowManifestKey: 'context-bundles' },
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
