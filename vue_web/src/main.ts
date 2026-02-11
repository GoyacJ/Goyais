/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import { createApp } from 'vue'
import App from './App.vue'
import i18n from '@/i18n'
import router from '@/router'
import { initDensitySystem } from '@/design-system/density'
import { initLayoutSystem } from '@/design-system/layout'
import { initThemeSystem } from '@/design-system/theme'
import './style.css'

initThemeSystem()
initDensitySystem()
initLayoutSystem(router)

createApp(App).use(router).use(i18n).mount('#app')
