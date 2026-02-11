<template>
  <section class="ui-page">
    <PageHeader :title="t('page.forbidden.title')" :subtitle="t('page.forbidden.subtitle')" />

    <WindowBoard route-key="forbidden" :panes="windowPanes">
      <template #forbidden-state>
        <EmptyState
          variant="forbidden-403"
          :title="t('empty_state.forbidden.title')"
          :description="t('empty_state.forbidden.description')"
        >
          <template #actions>
            <RouterLink to="/" class="ui-control ui-focus-ring ui-pressable ui-btn-secondary inline-flex items-center gap-2">
              <Icon name="home" :size="14" decorative />
              {{ t('common.backHome') }}
            </RouterLink>
          </template>
        </EmptyState>
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
import WindowBoard from '@/components/layout/WindowBoard.vue'
import Icon from '@/components/ui/Icon.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n({ useScope: 'global' })

const windowPanes = computed(() => [{ id: 'forbidden-state', title: t('empty_state.forbidden.title') }])
</script>
