import { createApp } from 'vue'
import App from './App.vue'
import i18n from '@/i18n'
import router from '@/router'
import { initDensitySystem } from '@/design-system/density'
import { initThemeSystem } from '@/design-system/theme'
import './style.css'

initThemeSystem()
initDensitySystem()

createApp(App).use(router).use(i18n).mount('#app')
