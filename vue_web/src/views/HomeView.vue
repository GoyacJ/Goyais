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

    <WindowBoard route-key="home" :panes="windowPanes">
      <template #design-tokens>
        <SectionCard :title="t('page.home.sections.designTokens.title')" :subtitle="t('page.home.sections.designTokens.subtitle')">
          <div class="grid grid-cols-4 gap-2">
            <span class="h-8 rounded-button border border-ui-border bg-ui-bg" />
            <span class="h-8 rounded-button border border-ui-border bg-ui-panel" />
            <span class="ui-tone-surface-primary h-8 rounded-button border" />
            <span class="ui-tone-surface-success h-8 rounded-button border" />
          </div>
        </SectionCard>
      </template>

      <template #state-hooks>
        <SectionCard :title="t('page.home.sections.stateHooks.title')" :subtitle="t('page.home.sections.stateHooks.subtitle')">
          <div class="flex flex-wrap gap-2">
            <Button variant="secondary">ui-focus-ring</Button>
            <Button variant="secondary" loading>ui-loading</Button>
            <Button variant="secondary" disabled>ui-disabled</Button>
          </div>
        </SectionCard>
      </template>

      <template #status>
        <SectionCard :title="t('page.home.sections.status.title')" :subtitle="t('page.home.sections.status.subtitle')">
          <div class="flex flex-wrap gap-2">
            <StatusBadge status="accepted" />
            <StatusBadge status="running" />
            <StatusBadge status="succeeded" />
            <StatusBadge status="failed" />
            <StatusBadge status="canceled" />
          </div>
        </SectionCard>
      </template>

      <template #backgrounds>
        <SectionCard :title="t('page.home.sections.backgrounds.title')" :subtitle="t('page.home.sections.backgrounds.subtitle')">
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
      </template>

      <template #empty-states>
        <SectionCard :title="t('page.home.sections.emptyStates.title')" :subtitle="t('page.home.sections.emptyStates.subtitle')">
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
      </template>
    </WindowBoard>
  </section>
</template>

<script setup lang="ts">
/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const windowPanes = computed(() => [
  { id: 'design-tokens', title: t('page.home.sections.designTokens.title') },
  { id: 'state-hooks', title: t('page.home.sections.stateHooks.title') },
  { id: 'status', title: t('page.home.sections.status.title') },
  { id: 'backgrounds', title: t('page.home.sections.backgrounds.title') },
  { id: 'empty-states', title: t('page.home.sections.emptyStates.title') },
])
</script>
