<template>
  <div class="space-y-3">
    <header class="flex flex-wrap items-center gap-2">
      <Input v-model="templateName" :placeholder="t('page.canvas.templateNamePlaceholder')" />
      <Button :disabled="busy || templateName.trim().length === 0" @click="onCreateTemplate">
        {{ t('page.canvas.actionCreateTemplate') }}
      </Button>
      <Button variant="secondary" :disabled="busy || !selectedTemplateId" @click="$emit('patch', selectedTemplateId as string)">
        {{ t('page.canvas.actionPatchTemplate') }}
      </Button>
      <Button variant="secondary" :disabled="busy || !selectedTemplateId" @click="$emit('publish', selectedTemplateId as string)">
        {{ t('page.canvas.actionPublishTemplate') }}
      </Button>
      <Button variant="ghost" :disabled="busy" @click="$emit('refresh')">{{ t('common.refresh') }}</Button>
    </header>

    <Table
      :columns="columns"
      :rows="rows"
      :state="state"
      :caption="t('page.canvas.templatesTitle')"
      interactive-rows
      row-key="id"
      :selected-row-key="selectedTemplateId"
      @row-click="onRowClick"
    />
  </div>
</template>

<script setup lang="ts">
import type { WorkflowTemplateDTO } from '@/api/types'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Table, { type TableColumn } from '@/components/ui/Table.vue'
import type { TableState } from '@/design-system/types'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    templates: WorkflowTemplateDTO[]
    selectedTemplateId: string | null
    busy?: boolean
    state?: TableState
  }>(),
  {
    busy: false,
    state: 'ready',
  },
)

const emit = defineEmits<{
  (e: 'select', templateId: string): void
  (e: 'create', payload: { name: string }): void
  (e: 'patch', templateId: string): void
  (e: 'publish', templateId: string): void
  (e: 'refresh'): void
}>()

const { t } = useI18n({ useScope: 'global' })

const templateName = ref('')

const columns = computed<TableColumn[]>(() => [
  { key: 'name', label: t('page.canvas.fieldTemplateName') },
  { key: 'status', label: t('page.canvas.fieldStatus'), width: '8rem' },
  { key: 'id', label: t('page.canvas.fieldTemplateId'), mono: true, width: '18rem' },
])

const rows = computed(() =>
  props.templates.map((item) => ({
    id: item.id,
    name: item.name,
    status: item.status,
  })),
)

function onCreateTemplate(): void {
  const name = templateName.value.trim()
  if (name.length === 0) {
    return
  }
  emit('create', { name })
  templateName.value = ''
}

function onRowClick(payload: { rowKey: string }): void {
  emit('select', payload.rowKey)
}
</script>
