<template>
  <section class="ui-page">
    <PageHeader :title="t('page.home.title')" :subtitle="t('page.home.subtitle')">
      <template #actions>
        <Button>
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <div class="grid gap-[var(--ui-page-gap)] lg:grid-cols-3">
      <SectionCard title="Design Tokens" subtitle="neutral / primary / semantic">
        <div class="grid grid-cols-4 gap-2">
          <span class="h-8 rounded-button border border-ui-border bg-ui-bg" />
          <span class="h-8 rounded-button border border-ui-border bg-ui-panel" />
          <span class="h-8 rounded-button bg-primary-600" />
          <span class="h-8 rounded-button bg-success" />
        </div>
      </SectionCard>

      <SectionCard title="State Hooks" subtitle="Material 3 semantics">
        <div class="flex flex-wrap gap-2">
          <Button variant="secondary">ui-focus-ring</Button>
          <Button variant="secondary" loading>ui-loading</Button>
          <Button variant="secondary" disabled>ui-disabled</Button>
        </div>
      </SectionCard>

      <SectionCard title="Status" subtitle="AI runtime states">
        <div class="flex flex-wrap gap-2">
          <StatusBadge status="accepted" />
          <StatusBadge status="running" />
          <StatusBadge status="succeeded" />
          <StatusBadge status="failed" />
          <StatusBadge status="canceled" />
        </div>
      </SectionCard>
    </div>

    <SectionCard title="Background Presets" subtitle="SVG-based reusable patterns for light/dark">
      <div class="grid gap-3 lg:grid-cols-3">
        <div class="ui-bg-host ui-bg-grid rounded-card border border-ui-border p-4">
          <div class="ui-bg-content text-sm font-medium">ui-bg-grid</div>
        </div>
        <div class="ui-bg-host ui-bg-gradient rounded-card border border-ui-border p-4">
          <div class="ui-bg-content text-sm font-medium">ui-bg-gradient</div>
        </div>
        <div class="ui-bg-host ui-bg-dots rounded-card border border-ui-border p-4">
          <div class="ui-bg-content text-sm font-medium">ui-bg-dots</div>
        </div>
      </div>
    </SectionCard>

    <SectionCard title="Empty States" subtitle="commands/assets/403/404/loading/error showcase">
      <div class="grid gap-3 xl:grid-cols-2">
        <EmptyState
          variant="commands-empty"
          :title="t('empty_state.commands.title')"
          :description="t('empty_state.commands.description')"
        />
        <EmptyState
          variant="assets-empty"
          :title="t('empty_state.assets.title')"
          :description="t('empty_state.assets.description')"
        />
        <EmptyState
          variant="forbidden-403"
          :title="t('empty_state.forbidden.title')"
          :description="t('empty_state.forbidden.description')"
        />
        <EmptyState
          variant="not-found-404"
          :title="t('empty_state.notFound.title')"
          :description="t('empty_state.notFound.description')"
        />
        <EmptyState
          variant="loading"
          :title="t('empty_state.loading.title')"
          :description="t('empty_state.loading.description')"
        />
        <EmptyState
          variant="error"
          :title="t('empty_state.error.title')"
          :description="t('empty_state.error.description')"
        />
      </div>
    </SectionCard>
  </section>
</template>

<script setup lang="ts">
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
</script>
