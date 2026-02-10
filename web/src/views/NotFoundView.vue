<template>
  <section class="ui-page">
    <PageHeader :title="t('page.notFound.title')" :subtitle="t('page.notFound.subtitle')" />
    <EmptyState
      variant="not-found-404"
      :title="t('empty_state.notFound.title')"
      :description="t('empty_state.notFound.description')"
    >
      <template #actions>
        <RouterLink to="/" class="ui-control ui-focus-ring ui-pressable ui-btn-secondary inline-flex items-center gap-2">
          <Icon name="home" :size="14" decorative />
          {{ t('common.backHome') }}
        </RouterLink>
      </template>
    </EmptyState>
  </section>
</template>

<script setup lang="ts">
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import Icon from '@/components/ui/Icon.vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n({ useScope: 'global' })
</script>
