<template>
  <section class="rounded-xl border border-slate-300 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
    <h2 class="text-xl font-semibold">{{ t('home.title') }}</h2>
    <p class="mt-2 text-sm text-slate-600 dark:text-slate-300">{{ t('home.subtitle') }}</p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
