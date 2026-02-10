<template>
  <section class="ui-page">
    <PageHeader :title="t('page.home.title')" :subtitle="t('page.home.subtitle')">
      <template #actions>
        <Button>{{ t('common.refresh') }}</Button>
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
  </section>
</template>

<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import Button from '@/components/ui/Button.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
</script>
