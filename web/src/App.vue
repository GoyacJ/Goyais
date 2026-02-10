<template>
  <main class="mx-auto min-h-screen w-full max-w-5xl p-6">
    <header class="mb-6 flex flex-wrap items-center justify-between gap-4 rounded-xl border border-slate-300 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900">
      <div>
        <h1 class="text-2xl font-bold">{{ t('app.title') }}</h1>
      </div>

      <div class="flex flex-wrap items-center gap-3 text-sm">
        <label class="flex items-center gap-2">
          <span>{{ t('app.language') }}</span>
          <select v-model="locale" class="rounded border border-slate-300 bg-transparent px-2 py-1 dark:border-slate-700">
            <option value="zh-CN">zh-CN</option>
            <option value="en-US">en-US</option>
          </select>
        </label>

        <label class="flex items-center gap-2">
          <span>{{ t('app.theme') }}</span>
          <select v-model="theme" class="rounded border border-slate-300 bg-transparent px-2 py-1 dark:border-slate-700">
            <option value="light">{{ t('app.light') }}</option>
            <option value="dark">{{ t('app.dark') }}</option>
          </select>
        </label>
      </div>
    </header>

    <nav class="mb-6 flex gap-4 text-sm">
      <RouterLink class="rounded border border-slate-300 px-3 py-2 hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800" to="/">
        {{ t('app.home') }}
      </RouterLink>
      <RouterLink class="rounded border border-slate-300 px-3 py-2 hover:bg-slate-100 dark:border-slate-700 dark:hover:bg-slate-800" to="/canvas">
        {{ t('app.canvas') }}
      </RouterLink>
    </nav>

    <RouterView />
  </main>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { RouterLink, RouterView } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

const theme = computed({
  get: () => (document.documentElement.classList.contains('dark') ? 'dark' : 'light'),
  set: (value: string) => {
    document.documentElement.classList.toggle('dark', value === 'dark')
  },
})

watch(
  () => locale.value,
  (value) => {
    localStorage.setItem('goyais.locale', value)
  },
  { immediate: true },
)

const storedLocale = localStorage.getItem('goyais.locale')
if (storedLocale === 'zh-CN' || storedLocale === 'en-US') {
  locale.value = storedLocale
}

const storedTheme = localStorage.getItem('goyais.theme')
if (storedTheme === 'light' || storedTheme === 'dark') {
  theme.value = storedTheme
} else {
  theme.value = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

watch(
  () => theme.value,
  (value) => {
    localStorage.setItem('goyais.theme', value)
  },
  { immediate: true },
)
</script>
