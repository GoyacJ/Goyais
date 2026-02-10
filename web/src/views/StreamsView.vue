<template>
  <section class="ui-page">
    <PageHeader :title="t('page.streams.title')" :subtitle="t('page.streams.subtitle')" />

    <WindowBoard route-key="streams" :panes="windowPanes">
      <template #stream-overview>
        <SectionCard title="Stream Overview" subtitle="State-focused shell only">
          <div class="grid gap-[var(--ui-page-gap)] lg:grid-cols-3">
            <article
              v-for="stream in streams"
              :key="stream.id"
              class="ui-surface p-4"
            >
              <p class="ui-monospace text-xs text-ui-muted">{{ stream.id }}</p>
              <p class="mt-1 text-sm font-semibold">{{ stream.path }}</p>
              <div class="mt-3">
                <StatusBadge :status="stream.status" />
              </div>
            </article>
          </div>
        </SectionCard>
      </template>

      <template #stream-logs>
        <SectionCard title="Stream Logs" subtitle="Monospace readability baseline">
          <pre class="ui-monospace rounded-button border border-ui-border bg-ui-panel p-3 text-xs text-ui-muted">[10:44:01] stream connected
[10:44:04] recording started
[10:44:20] segment flushed</pre>
        </SectionCard>
      </template>
    </WindowBoard>
  </section>
</template>

<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import type { CommandStatus } from '@/design-system/types'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const windowPanes = computed(() => [
  { id: 'stream-overview', title: 'Stream Overview' },
  { id: 'stream-logs', title: 'Stream Logs' },
])

const streams: Array<{ id: string; path: string; status: CommandStatus }> = [
  { id: 'str_01', path: '/live/cam-01', status: 'running' },
  { id: 'str_02', path: '/live/cam-02', status: 'accepted' },
  { id: 'str_03', path: '/live/cam-03', status: 'canceled' },
]
</script>
