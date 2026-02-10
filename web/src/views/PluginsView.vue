<template>
  <section class="ui-page">
    <PageHeader :title="t('page.plugins.title')" :subtitle="t('page.plugins.subtitle')" />

    <SectionCard title="Plugin Catalog" subtitle="Card layout + semantic tags">
      <div class="grid gap-[var(--ui-page-gap)] md:grid-cols-2">
        <article
          v-for="plugin in plugins"
          :key="plugin.name"
          class="ui-surface p-4"
        >
          <header class="flex items-start justify-between gap-2">
            <div>
              <p class="text-sm font-semibold">{{ plugin.name }}</p>
              <p class="ui-monospace mt-1 text-xs text-ui-muted">{{ plugin.version }}</p>
            </div>
            <Badge :tone="plugin.enabled ? 'success' : 'warn'">
              {{ plugin.enabled ? 'enabled' : 'disabled' }}
            </Badge>
          </header>

          <p class="mt-3 text-sm text-ui-muted">{{ plugin.description }}</p>
          <div class="mt-3 flex gap-2">
            <Button variant="secondary">Enable</Button>
            <Button variant="ghost">Disable</Button>
          </div>
        </article>
      </div>
    </SectionCard>
  </section>
</template>

<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const plugins = [
  {
    name: 'MediaToolkit',
    version: 'v0.1.4',
    enabled: true,
    description: 'Streaming helpers and recording presets.',
  },
  {
    name: 'PromptGuard',
    version: 'v0.0.9',
    enabled: false,
    description: 'Input normalization and policy checks.',
  },
]
</script>
