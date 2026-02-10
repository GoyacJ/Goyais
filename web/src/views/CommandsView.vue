<template>
  <section class="ui-page">
    <PageHeader :title="t('page.commands.title')" :subtitle="t('page.commands.subtitle')">
      <template #actions>
        <Button>{{ t('common.refresh') }}</Button>
      </template>
    </PageHeader>

    <ErrorBanner :error="demoError" />

    <div class="grid gap-[var(--ui-page-gap)] xl:grid-cols-[1.25fr_1fr]">
      <SectionCard title="Command Feed" subtitle="commandId / status / timestamps">
        <div class="space-y-3">
          <CommandCard v-for="item in mockCommands" :key="item.commandId" :command="item" />
        </div>
      </SectionCard>

      <SectionCard title="Logs" subtitle="Collapsible monospace panel">
        <LogPanel :lines="activeLogs" />
      </SectionCard>
    </div>

    <SectionCard title="List Shape" subtitle="Table + Pagination UI shell">
      <Table :columns="columns" :rows="rows" />
      <div class="mt-3">
        <Pagination :page="page" :page-size="10" :total="40" @update:page="page = $event" />
      </div>
    </SectionCard>
  </section>
</template>

<script setup lang="ts">
import ErrorBanner from '@/components/layout/ErrorBanner.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import CommandCard from '@/components/runtime/CommandCard.vue'
import LogPanel from '@/components/runtime/LogPanel.vue'
import Button from '@/components/ui/Button.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Table from '@/components/ui/Table.vue'
import { mockCommands } from '@/mocks/commands'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const page = ref(1)

const demoError = {
  code: 'FORBIDDEN',
  messageKey: 'error.authz.forbidden',
  details: {},
}

const columns = [
  { key: 'commandId', label: 'commandId', mono: true },
  { key: 'commandType', label: 'commandType' },
  { key: 'status', label: 'status' },
]

const rows = mockCommands.map((item) => ({
  commandId: item.commandId,
  commandType: item.commandType,
  status: item.status,
}))

const activeLogs = computed(() => mockCommands[0]?.logs ?? [])
</script>
