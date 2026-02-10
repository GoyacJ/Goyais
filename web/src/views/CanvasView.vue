<template>
  <section class="ui-page">
    <PageHeader :title="t('page.canvas.title')" :subtitle="t('page.canvas.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isBusy" @click="onRefreshAll">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <div v-if="apiError" class="ui-surface mb-3 p-3 text-sm text-ui-danger">
      {{ t(apiError.messageKey || 'error.common.internal', apiError.details ?? {}) }}
    </div>

    <WindowBoard route-key="canvas" :panes="windowPanes">
      <template #canvas-templates>
        <SectionCard :title="t('page.canvas.templatesTitle')" :subtitle="String(templates.length)">
          <WorkflowTemplatePane
            :templates="templates"
            :selected-template-id="selectedTemplateId"
            :busy="isBusy"
            :state="templateTableState"
            @select="onSelectTemplate"
            @create="onCreateTemplate"
            @patch="onPatchTemplate"
            @publish="onPublishTemplate"
            @refresh="loadTemplates"
          />
        </SectionCard>
      </template>

      <template #canvas-runs>
        <SectionCard :title="t('page.canvas.runsTitle')" :subtitle="String(runs.length)">
          <WorkflowRunPane
            :runs="runs"
            :selected-run-id="selectedRunId"
            :selected-template-id="selectedTemplateId"
            :busy="isBusy"
            :state="runTableState"
            @select="onSelectRun"
            @create="onCreateRun"
            @cancel="onCancelRun"
            @refresh="loadRuns"
          />
        </SectionCard>
      </template>

      <template #canvas-steps>
        <SectionCard :title="t('page.canvas.stepsTitle')" :subtitle="selectedRunId ?? '-'">
          <WorkflowStepPane :run-id="selectedRunId" :steps="steps" :loading="stepsLoading" />
        </SectionCard>
      </template>

      <template #canvas-registry>
        <SectionCard :title="t('page.canvas.registryTitle')" :subtitle="t('page.canvas.registrySubtitle')">
          <RegistryPane
            :capabilities="capabilities"
            :algorithms="algorithms"
            :providers="providers"
            :busy="isBusy"
            :loading="registryLoading"
            @run-algorithm="onRunAlgorithm"
            @refresh="loadRegistry"
          />
        </SectionCard>
      </template>
    </WindowBoard>
  </section>
</template>

<script setup lang="ts">
import { runAlgorithm } from '@/api/algorithms'
import { ApiHttpError, isMockEnabled } from '@/api/http'
import { listAlgorithms, listCapabilities, listProviders } from '@/api/registry'
import type {
  AlgorithmDTO,
  ApiError,
  CapabilityDTO,
  ProviderDTO,
  StepRunDTO,
  WorkflowRunDTO,
  WorkflowTemplateDTO,
} from '@/api/types'
import {
  cancelWorkflowRun,
  createWorkflowRun,
  createWorkflowTemplate,
  listWorkflowRuns,
  listWorkflowStepRuns,
  listWorkflowTemplates,
  patchWorkflowTemplate,
  publishWorkflowTemplate,
} from '@/api/workflow'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import RegistryPane from '@/components/runtime/RegistryPane.vue'
import WorkflowRunPane from '@/components/runtime/WorkflowRunPane.vue'
import WorkflowStepPane from '@/components/runtime/WorkflowStepPane.vue'
import WorkflowTemplatePane from '@/components/runtime/WorkflowTemplatePane.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import { useToast } from '@/composables/useToast'
import type { TableState } from '@/design-system/types'
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const useMock = isMockEnabled()

const templates = ref<WorkflowTemplateDTO[]>([])
const runs = ref<WorkflowRunDTO[]>([])
const steps = ref<StepRunDTO[]>([])
const capabilities = ref<CapabilityDTO[]>([])
const algorithms = ref<AlgorithmDTO[]>([])
const providers = ref<ProviderDTO[]>([])

const selectedTemplateId = ref<string | null>(null)
const selectedRunId = ref<string | null>(null)

const templatesLoading = ref(false)
const runsLoading = ref(false)
const stepsLoading = ref(false)
const registryLoading = ref(false)
const isBusy = ref(false)
const apiError = ref<ApiError | null>(null)

const windowPanes = computed(() => [
  { id: 'canvas-templates', title: t('page.canvas.templatesTitle') },
  { id: 'canvas-runs', title: t('page.canvas.runsTitle') },
  { id: 'canvas-steps', title: t('page.canvas.stepsTitle') },
  { id: 'canvas-registry', title: t('page.canvas.registryTitle') },
])

const templateTableState = computed<TableState>(() => {
  if (templatesLoading.value) {
    return 'loading'
  }
  return templates.value.length > 0 ? 'ready' : 'empty'
})

const runTableState = computed<TableState>(() => {
  if (runsLoading.value) {
    return 'loading'
  }
  return runs.value.length > 0 ? 'ready' : 'empty'
})

watch(
  templates,
  (items) => {
    if (!items.some((item) => item.id === selectedTemplateId.value)) {
      selectedTemplateId.value = items[0]?.id ?? null
    }
  },
  { immediate: true },
)

watch(
  runs,
  (items) => {
    if (!items.some((item) => item.id === selectedRunId.value)) {
      selectedRunId.value = items[0]?.id ?? null
    }
  },
  { immediate: true },
)

watch(selectedRunId, (runId) => {
  if (!runId) {
    steps.value = []
    return
  }
  void loadSteps(runId)
})

onMounted(() => {
  void onRefreshAll()
})

async function onRefreshAll(): Promise<void> {
  apiError.value = null
  await Promise.all([loadTemplates(), loadRuns(), loadRegistry()])
}

async function loadTemplates(): Promise<void> {
  templatesLoading.value = true
  try {
    if (useMock) {
      templates.value = []
      return
    }
    const response = await listWorkflowTemplates({ page: 1, pageSize: 200 })
    templates.value = response.items
  } catch (error) {
    apiError.value = asApiError(error)
    templates.value = []
  } finally {
    templatesLoading.value = false
  }
}

async function loadRuns(): Promise<void> {
  runsLoading.value = true
  try {
    if (useMock) {
      runs.value = []
      return
    }
    const response = await listWorkflowRuns({ page: 1, pageSize: 200 })
    runs.value = response.items
  } catch (error) {
    apiError.value = asApiError(error)
    runs.value = []
  } finally {
    runsLoading.value = false
  }
}

async function loadRegistry(): Promise<void> {
  registryLoading.value = true
  try {
    if (useMock) {
      capabilities.value = []
      algorithms.value = []
      providers.value = []
      return
    }
    const [capabilityResp, algorithmResp, providerResp] = await Promise.all([
      listCapabilities({ page: 1, pageSize: 200 }),
      listAlgorithms({ page: 1, pageSize: 200 }),
      listProviders({ page: 1, pageSize: 200 }),
    ])
    capabilities.value = capabilityResp.items
    algorithms.value = algorithmResp.items
    providers.value = providerResp.items
  } catch (error) {
    apiError.value = asApiError(error)
    capabilities.value = []
    algorithms.value = []
    providers.value = []
  } finally {
    registryLoading.value = false
  }
}

async function loadSteps(runId: string): Promise<void> {
  stepsLoading.value = true
  try {
    if (useMock) {
      steps.value = []
      return
    }
    const response = await listWorkflowStepRuns(runId, { page: 1, pageSize: 200 })
    steps.value = response.items
  } catch (error) {
    apiError.value = asApiError(error)
    steps.value = []
  } finally {
    stepsLoading.value = false
  }
}

function onSelectTemplate(templateId: string): void {
  selectedTemplateId.value = templateId
}

function onSelectRun(runId: string): void {
  selectedRunId.value = runId
}

async function onCreateTemplate(payload: { name: string }): Promise<void> {
  if (isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    const response = await createWorkflowTemplate({
      name: payload.name,
      description: 'Created from canvas runtime MVP',
      graph: { nodes: [{ id: 'step-1', type: 'noop' }], edges: [] },
      schemaInputs: {},
      schemaOutputs: {},
      visibility: 'PRIVATE',
    })
    pushToast({
      title: t('page.canvas.actionCreateTemplate'),
      message: `${t('page.canvas.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadTemplates()
    selectedTemplateId.value = response.resource.id
  } catch (error) {
    notifyActionError(t('page.canvas.actionCreateTemplate'), error)
  } finally {
    isBusy.value = false
  }
}

async function onPatchTemplate(templateId: string): Promise<void> {
  if (isBusy.value || useMock) {
    return
  }
  const template = templates.value.find((item) => item.id === templateId)
  if (!template) {
    return
  }

  isBusy.value = true
  try {
    const nextGraph = { ...template.graph, patchedAt: new Date().toISOString() }
    const response = await patchWorkflowTemplate(templateId, {
      graph: nextGraph,
    })
    pushToast({
      title: t('page.canvas.actionPatchTemplate'),
      message: `${t('page.canvas.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadTemplates()
  } catch (error) {
    notifyActionError(t('page.canvas.actionPatchTemplate'), error)
  } finally {
    isBusy.value = false
  }
}

async function onPublishTemplate(templateId: string): Promise<void> {
  if (isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    const response = await publishWorkflowTemplate(templateId)
    pushToast({
      title: t('page.canvas.actionPublishTemplate'),
      message: `${t('page.canvas.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadTemplates()
  } catch (error) {
    notifyActionError(t('page.canvas.actionPublishTemplate'), error)
  } finally {
    isBusy.value = false
  }
}

async function onCreateRun(payload: { mode: 'sync' | 'running' | 'fail' }): Promise<void> {
  if (isBusy.value || useMock) {
    return
  }
  if (!selectedTemplateId.value) {
    return
  }
  isBusy.value = true
  try {
    const response = await createWorkflowRun({
      templateId: selectedTemplateId.value,
      inputs: { source: 'web.canvas.runtime' },
      mode: payload.mode,
    })
    pushToast({
      title: t('page.canvas.actionRunTemplate'),
      message: `${t('page.canvas.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadRuns()
    selectedRunId.value = response.resource.id
  } catch (error) {
    notifyActionError(t('page.canvas.actionRunTemplate'), error)
  } finally {
    isBusy.value = false
  }
}

async function onCancelRun(runId: string): Promise<void> {
  if (isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    const response = await cancelWorkflowRun(runId)
    pushToast({
      title: t('page.canvas.actionCancelRun'),
      message: `${t('page.canvas.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadRuns()
    await loadSteps(runId)
  } catch (error) {
    notifyActionError(t('page.canvas.actionCancelRun'), error)
  } finally {
    isBusy.value = false
  }
}

async function onRunAlgorithm(algorithmId: string): Promise<void> {
  if (isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    const response = await runAlgorithm(algorithmId, {
      inputs: { source: 'web.canvas.registry' },
      mode: 'sync',
    })
    pushToast({
      title: t('page.canvas.actionRunAlgorithm'),
      message: `${t('page.canvas.fieldCommandId')}: ${response.commandRef.commandId}`,
      tone: 'success',
    })
    await loadRuns()
    if (response.resource.workflowRunId) {
      selectedRunId.value = response.resource.workflowRunId
    }
  } catch (error) {
    notifyActionError(t('page.canvas.actionRunAlgorithm'), error)
  } finally {
    isBusy.value = false
  }
}

function notifyActionError(title: string, error: unknown): void {
  const apiErr = asApiError(error)
  pushToast({
    title,
    message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    tone: 'error',
  })
}

function asApiError(value: unknown): ApiError {
  if (value instanceof ApiHttpError) {
    return value.error
  }
  return {
    code: 'INTERNAL_ERROR',
    messageKey: 'error.common.internal',
  }
}
</script>
