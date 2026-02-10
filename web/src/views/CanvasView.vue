<template>
  <section class="ui-page">
    <PageHeader :title="t('page.canvas.title')" :subtitle="t('page.canvas.subtitle')" />

    <SectionCard title="Canvas Surface" subtitle="Node radius constrained to 6-8px">
      <div class="grid min-h-80 place-items-center rounded-card border border-dashed border-ui-border bg-ui-bg p-4">
        <div class="grid w-full max-w-2xl grid-cols-3 gap-3">
          <div
            v-for="node in nodes"
            :key="node"
            class="rounded-canvas border border-ui-border bg-ui-panel p-3 text-sm shadow-none"
          >
            <p class="ui-monospace text-xs text-ui-muted">node_{{ node }}</p>
            <p class="mt-1 font-medium">Operator {{ node }}</p>
          </div>
        </div>
      </div>
    </SectionCard>
  </section>
</template>

<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const nodes = ['A1', 'B2', 'C3', 'D4', 'E5', 'F6']
</script>
