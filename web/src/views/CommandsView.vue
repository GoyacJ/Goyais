<template>
  <section class="ui-page">
    <PageHeader :title="t('page.commands.title')" :subtitle="t('page.commands.subtitle')">
      <template #actions>
        <Button variant="secondary">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
        <Button>
          <Icon name="commands" :size="14" decorative />
          {{ t('page.commands.actionRun') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="commands" :panes="windowPanes">
      <template #filters>
        <SectionCard :title="t('page.commands.filtersTitle')" :subtitle="t('page.commands.filtersSubtitle')">
          <div class="grid gap-3 md:grid-cols-[1.4fr_1fr_1fr]">
            <Input v-model="searchQuery" :placeholder="t('page.commands.searchPlaceholder')" />
            <Select v-model="statusFilter" :options="statusOptions" />
            <Select v-model="ownerFilter" :options="ownerOptions" />
          </div>
        </SectionCard>
      </template>

      <template #list>
        <SectionCard :title="t('page.commands.listTitle')" :subtitle="`${filteredCommands.length}`">
          <div v-if="filteredCommands.length === 0">
            <EmptyState
              variant="commands-empty"
              :title="t('empty_state.commands.title')"
              :description="t('empty_state.commands.description')"
            />
          </div>
          <div v-else class="space-y-2">
            <CommandCard
              v-for="item in filteredCommands"
              :key="item.commandId"
              :command="item"
              interactive
              :selected="selectedCommand?.commandId === item.commandId"
              @select="selectedCommandId = $event"
            />
          </div>
        </SectionCard>
      </template>

      <template #detail>
        <SectionCard :title="t('page.commands.detailTitle')" :subtitle="selectedCommand?.commandId ?? '-'">
          <div v-if="selectedCommand" class="space-y-3">
            <Tabs v-model="detailTab" :tabs="detailTabs" :aria-label="t('page.commands.detailTitle')" />

            <div v-if="detailTab === 'summary'" class="ui-surface border-ui-borderSubtle bg-ui-surface2 p-3">
              <dl class="grid gap-3 text-xs text-ui-muted md:grid-cols-2">
                <div>
                  <dt>{{ t('page.commands.fieldType') }}</dt>
                  <dd class="mt-1 text-sm text-ui-fg">{{ selectedCommand.commandType }}</dd>
                </div>
                <div>
                  <dt>{{ t('page.commands.fieldStatus') }}</dt>
                  <dd class="mt-1"><StatusBadge :status="selectedCommand.status" /></dd>
                </div>
                <div>
                  <dt>{{ t('page.commands.fieldAcceptedAt') }}</dt>
                  <dd class="ui-monospace mt-1 text-ui-fg">{{ selectedCommand.acceptedAt }}</dd>
                </div>
                <div>
                  <dt>{{ t('page.commands.fieldOwner') }}</dt>
                  <dd class="ui-monospace mt-1 text-ui-fg">{{ selectedCommand.owner }}</dd>
                </div>
                <div class="md:col-span-2">
                  <dt>{{ t('page.commands.fieldResult') }}</dt>
                  <dd class="mt-1 text-sm text-ui-fg">{{ selectedCommand.resultSummary }}</dd>
                </div>
              </dl>
            </div>

            <LogPanel v-else :lines="selectedCommand.logs" />
          </div>
          <EmptyState
            v-else
            variant="commands-empty"
            :title="t('empty_state.commands.detailTitle')"
            :description="t('empty_state.commands.detailDescription')"
          />
        </SectionCard>
      </template>
    </WindowBoard>
  </section>
</template>

<script setup lang="ts">
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import CommandCard from '@/components/runtime/CommandCard.vue'
import StatusBadge from '@/components/runtime/StatusBadge.vue'
import LogPanel from '@/components/runtime/LogPanel.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Tabs from '@/components/ui/Tabs.vue'
import { mockCommands } from '@/mocks/commands'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })

const searchQuery = ref('')
const statusFilter = ref('all')
const ownerFilter = ref('all')
const selectedCommandId = ref<string | null>(mockCommands[0]?.commandId ?? null)
const detailTab = ref('summary')

const windowPanes = computed(() => [
  { id: 'filters', title: t('page.commands.filtersTitle') },
  { id: 'list', title: t('page.commands.listTitle') },
  { id: 'detail', title: t('page.commands.detailTitle') },
])

const statusOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  { value: 'accepted', label: t('status.accepted') },
  { value: 'running', label: t('status.running') },
  { value: 'succeeded', label: t('status.succeeded') },
  { value: 'failed', label: t('status.failed') },
  { value: 'canceled', label: t('status.canceled') },
])

const ownerOptions = computed(() => [
  { value: 'all', label: t('common.all') },
  ...Array.from(new Set(mockCommands.map((item) => item.owner))).map((owner) => ({
    value: owner,
    label: owner,
  })),
])

const filteredCommands = computed(() =>
  mockCommands.filter((item) => {
    const matchStatus = statusFilter.value === 'all' || item.status === statusFilter.value
    const matchOwner = ownerFilter.value === 'all' || item.owner === ownerFilter.value
    const q = searchQuery.value.trim().toLowerCase()
    const matchQuery =
      q.length === 0 ||
      item.commandId.toLowerCase().includes(q) ||
      item.commandType.toLowerCase().includes(q) ||
      item.traceId.toLowerCase().includes(q)

    return matchStatus && matchOwner && matchQuery
  }),
)

watch(
  filteredCommands,
  (items) => {
    if (!items.some((item) => item.commandId === selectedCommandId.value)) {
      selectedCommandId.value = items[0]?.commandId ?? null
    }
  },
  { immediate: true },
)

const selectedCommand = computed(() =>
  filteredCommands.value.find((item) => item.commandId === selectedCommandId.value),
)

const detailTabs = computed(() => [
  { id: 'summary', label: t('page.commands.tabSummary') },
  { id: 'logs', label: t('page.commands.tabLogs') },
])
</script>
