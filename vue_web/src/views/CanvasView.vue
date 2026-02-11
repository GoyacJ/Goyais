<template>
  <section class="ui-page">
    <PageHeader :title="t('page.canvas.title')" :subtitle="t('page.canvas.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isBusy || templatesLoading || runsLoading" @click="onRefreshAll">
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
          <div class="space-y-3">
          <div class="flex gap-2">
            <Input v-model="templateName" :placeholder="t('page.canvas.templateNamePlaceholder')" />
            <Button :disabled="isBusy || templateName.trim().length === 0" @click="onCreateTemplate">
              {{ t('page.canvas.actionCreateTemplate') }}
            </Button>
          </div>

          <div class="space-y-2">
            <button
              v-for="item in templates"
              :key="item.id"
              type="button"
              class="w-full rounded-lg border px-3 py-2 text-left text-sm transition"
              :class="
                item.id === selectedTemplateId
                  ? 'border-ui-primary bg-ui-primary/10 text-ui-text'
                  : 'border-ui-border bg-ui-panel text-ui-text hover:border-ui-primary/60'
              "
              @click="onSelectTemplate(item.id)"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="font-medium">{{ item.name }}</span>
                <span class="text-xs uppercase text-ui-muted">{{ item.status }}</span>
              </div>
              <div class="mt-1 text-xs text-ui-muted">{{ item.id }}</div>
            </button>
          </div>

          <div class="grid grid-cols-2 gap-2">
            <Button variant="secondary" :disabled="!selectedTemplateId || isBusy" @click="onPatchTemplate">
              {{ t('page.canvas.actionPatchTemplate') }}
            </Button>
            <Button variant="secondary" :disabled="!selectedTemplateId || isBusy" @click="onPublishTemplate">
              {{ t('page.canvas.actionPublishTemplate') }}
            </Button>
          </div>

          <div class="border-t border-ui-border pt-3">
            <div class="mb-2 text-sm font-medium text-ui-text">{{ t('page.canvas.runsTitle') }}</div>
            <div class="space-y-2">
              <button
                v-for="run in runs"
                :key="run.id"
                type="button"
                class="w-full rounded-lg border px-3 py-2 text-left text-xs transition"
                :class="
                  run.id === selectedRunId
                    ? 'border-ui-primary bg-ui-primary/10 text-ui-text'
                    : 'border-ui-border bg-ui-panel text-ui-text hover:border-ui-primary/60'
                "
                @click="onSelectRun(run.id)"
              >
                <div class="flex items-center justify-between gap-2">
                  <span class="font-medium">{{ run.id }}</span>
                  <span class="uppercase text-ui-muted">{{ run.status }}</span>
                </div>
                <div class="mt-1 text-ui-muted">{{ run.templateId }}</div>
              </button>
            </div>
          </div>
          </div>
        </SectionCard>
      </template>

      <template #canvas-board>
        <SectionCard :title="t('page.canvas.boardTitle')" :subtitle="selectedTemplateId ?? '-'">
          <div class="space-y-3">
          <div class="grid grid-cols-2 gap-2 md:grid-cols-4">
            <Button :disabled="!selectedTemplateId || isBusy" @click="onRunTemplate">
              {{ t('page.canvas.actionRunTemplate') }}
            </Button>
            <Button variant="secondary" :disabled="!selectedTemplateId || !selectedNodeId || isBusy" @click="onRunFromHere">
              {{ t('page.canvas.actionRunFromHere') }}
            </Button>
            <Button variant="secondary" :disabled="!selectedTemplateId || !selectedNodeId || isBusy" @click="onTestNode">
              {{ t('page.canvas.actionTestNode') }}
            </Button>
            <Button variant="ghost" :disabled="!selectedRunId || isBusy" @click="onCancelRun">
              {{ t('page.canvas.actionCancelRun') }}
            </Button>
          </div>

          <div class="flex flex-wrap items-center gap-2 rounded-lg border border-ui-border bg-ui-panel px-3 py-2 text-xs text-ui-muted">
            <span>{{ t('page.canvas.history') }}</span>
            <Button variant="secondary" :disabled="!canUndo" @click="onUndo">{{ t('page.canvas.undo') }}</Button>
            <Button variant="secondary" :disabled="!canRedo" @click="onRedo">{{ t('page.canvas.redo') }}</Button>
            <span class="ml-auto">{{ t('page.canvas.selectedNode') }}: {{ selectedNodeId ?? '-' }}</span>
          </div>

          <div class="grid gap-2 md:grid-cols-2">
            <Button
              v-for="preset in nodePresets"
              :key="preset.nodeType"
              variant="ghost"
              class="justify-start"
              :disabled="!selectedTemplateId || isBusy"
              @click="onAddNode(preset)"
            >
              {{ preset.label }} ({{ preset.inputType }} → {{ preset.outputType }})
            </Button>
          </div>

          <div class="h-[560px] overflow-hidden rounded-xl border border-ui-border bg-ui-bg">
            <VueFlow
              v-model:nodes="graphNodes"
              v-model:edges="graphEdges"
              class="h-full w-full"
              fit-view-on-init
              :min-zoom="0.2"
              :max-zoom="2"
              :node-types="nodeTypes"
              :default-edge-options="{ animated: false }"
              :connection-mode="ConnectionMode.Loose"
              @connect="onConnect"
              @node-click="onNodeClick"
            >
              <Background pattern-color="rgba(140, 155, 175, 0.2)" :gap="24" />
              <MiniMap pannable zoomable />
              <Controls />
            </VueFlow>
          </div>
          </div>
        </SectionCard>
      </template>

      <template #canvas-inspector>
        <SectionCard :title="t('page.canvas.inspectorTitle')" :subtitle="selectedNodeId ?? '-'">
          <div class="space-y-3">
          <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-xs text-ui-muted">
            <div>{{ t('page.canvas.connectionRule') }}</div>
            <div class="mt-1 text-ui-text">{{ t('page.canvas.connectionRuleHint') }}</div>
            <div v-if="connectionError" class="mt-2 text-ui-danger">{{ connectionError }}</div>
          </div>

          <div
            v-if="selectedNode && selectedNodeData"
            class="space-y-2 rounded-lg border border-ui-border bg-ui-panel p-3 text-xs text-ui-text"
          >
            <div class="font-medium">{{ selectedNodeData.label }}</div>
            <div>{{ t('page.canvas.nodeType') }}: {{ selectedNodeData.nodeType }}</div>
            <div>in: {{ selectedNodeData.inputType }}</div>
            <div>out: {{ selectedNodeData.outputType }}</div>
            <Button variant="destructive" :disabled="isBusy" @click="onRemoveSelectedNode">
              {{ t('page.canvas.actionRemoveNode') }}
            </Button>
          </div>

          <div
            v-if="selectedNodeRuntime"
            class="canvas-node-runtime space-y-1 rounded-lg border border-ui-border bg-ui-panel p-3 text-xs text-ui-text"
          >
            <div class="font-medium">{{ t('page.canvas.runtimeTitle') }}</div>
            <div>{{ t('page.canvas.fieldStatus') }}: {{ selectedNodeRuntime.status }}</div>
            <div v-if="typeof selectedNodeRuntime.durationMs === 'number'">
              {{ t('page.canvas.fieldDurationMs') }}: {{ selectedNodeRuntime.durationMs }}
            </div>
            <div>{{ t('page.canvas.runtimeArtifacts') }}: {{ selectedNodeRuntime.artifactCount }}</div>
            <div v-if="selectedNodeRuntime.logRef">{{ t('page.canvas.runtimeLogRef') }}: {{ selectedNodeRuntime.logRef }}</div>
            <div v-if="selectedNodeRuntime.errorCode" class="text-ui-danger">
              {{ t('page.canvas.runtimeErrorCode') }}: {{ selectedNodeRuntime.errorCode }}
            </div>
          </div>

          <div class="rounded-lg border border-ui-border bg-ui-panel p-3 text-xs">
            <div class="mb-2 font-medium text-ui-text">{{ t('page.canvas.patchDiffTitle') }}</div>
            <div class="space-y-1 text-ui-muted">
              <div>{{ t('page.canvas.patchAddedNodes') }}: {{ patchDiff.addedNodes.length }}</div>
              <div>{{ t('page.canvas.patchRemovedNodes') }}: {{ patchDiff.removedNodes.length }}</div>
              <div>{{ t('page.canvas.patchChangedNodes') }}: {{ patchDiff.changedNodes.length }}</div>
              <div>{{ t('page.canvas.patchAddedEdges') }}: {{ patchDiff.addedEdges.length }}</div>
              <div>{{ t('page.canvas.patchRemovedEdges') }}: {{ patchDiff.removedEdges.length }}</div>
            </div>
          </div>

          <div class="rounded-lg border border-ui-border bg-ui-panel p-3">
            <div class="mb-2 text-xs font-medium text-ui-text">{{ t('page.canvas.stepsTitle') }}</div>
            <div class="space-y-1 text-xs text-ui-muted">
              <button
                v-for="step in steps"
                :key="step.id"
                type="button"
                class="canvas-step-item w-full rounded border px-2 py-1 text-left transition"
                :class="
                  step.stepKey === selectedNodeId
                    ? 'border-ui-primary bg-ui-primary/10 text-ui-text'
                    : 'border-ui-border bg-ui-panel text-ui-muted hover:border-ui-primary/60'
                "
                @click="onSelectStep(step.stepKey)"
              >
                <div class="flex items-center justify-between gap-2">
                  <span>{{ step.stepKey }}</span>
                  <span class="uppercase">{{ step.status }}</span>
                </div>
                <div class="mt-1 text-[11px]">
                  <span v-if="typeof step.durationMs === 'number'">{{ step.durationMs }}ms</span>
                  <span v-if="typeof step.durationMs === 'number'" class="px-1">·</span>
                  <span>{{ t('page.canvas.runtimeArtifacts') }}: {{ countArtifacts(step.artifacts) }}</span>
                </div>
              </button>
              <div v-if="steps.length === 0">{{ t('page.canvas.stepsEmptyDescription') }}</div>
            </div>
          </div>
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
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

import { ApiHttpError, isMockEnabled } from '@/api/http'
import type { ApiError, StepRunDTO, WorkflowRunDTO, WorkflowTemplateDTO } from '@/api/types'
import {
  cancelWorkflowRun,
  createWorkflowRun,
  createWorkflowTemplate,
  getWorkflowRun,
  getWorkflowTemplate,
  listWorkflowRuns,
  listWorkflowStepRuns,
  listWorkflowTemplates,
  patchWorkflowTemplate,
  publishWorkflowTemplate,
} from '@/api/workflow'
import { type CanvasStepRuntime, canvasStepRuntimeByKeyKey } from '@/components/canvas/runtime'
import TypedPortNode from '@/components/canvas/TypedPortNode.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { useToast } from '@/composables/useToast'
import {
  ConnectionMode,
  VueFlow,
  type Connection,
  type Edge,
  type Node,
} from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

type GraphNodeData = {
  label: string
  inputType: string
  outputType: string
  nodeType: string
}

type GraphSnapshot = {
  nodes: Node<GraphNodeData>[]
  edges: Edge[]
}

type NodePreset = {
  nodeType: string
  label: string
  inputType: string
  outputType: string
}

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const useMock = isMockEnabled()
const nodeTypes = { typed: TypedPortNode }

const nodePresets: NodePreset[] = [
  { nodeType: 'source.text', label: 'Text Source', inputType: 'none', outputType: 'text' },
  { nodeType: 'transform.json', label: 'JSON Transform', inputType: 'text', outputType: 'json' },
  { nodeType: 'tool.http', label: 'HTTP Tool', inputType: 'json', outputType: 'json' },
  { nodeType: 'sink.asset', label: 'Asset Sink', inputType: 'json', outputType: 'any' },
]

const templates = ref<WorkflowTemplateDTO[]>([])
const runs = ref<WorkflowRunDTO[]>([])
const steps = ref<StepRunDTO[]>([])
const graphNodes = ref<Node<GraphNodeData>[]>([])
const graphEdges = ref<Edge[]>([])
const baseGraph = ref<GraphSnapshot>({ nodes: [], edges: [] })

const selectedTemplateId = ref<string | null>(null)
const selectedRunId = ref<string | null>(null)
const selectedNodeId = ref<string | null>(null)

const templateName = ref('')
const templatesLoading = ref(false)
const runsLoading = ref(false)
const stepsLoading = ref(false)
const isBusy = ref(false)
const apiError = ref<ApiError | null>(null)
const connectionError = ref('')

const historyStack = ref<string[]>([])
const historyIndex = ref(-1)
const restoringHistory = ref(false)
const runRuntimeTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const runRuntimePolling = ref(false)

const canUndo = computed(() => historyIndex.value > 0)
const canRedo = computed(() => historyIndex.value >= 0 && historyIndex.value < historyStack.value.length - 1)
const windowPanes = computed(() => [
  { id: 'canvas-templates', title: t('page.canvas.templatesTitle') },
  { id: 'canvas-board', title: t('page.canvas.boardTitle') },
  { id: 'canvas-inspector', title: t('page.canvas.inspectorTitle') },
])

const selectedNode = computed(() => graphNodes.value.find((node) => node.id === selectedNodeId.value) ?? null)
const selectedNodeData = computed<GraphNodeData | null>(() => {
  const node = selectedNode.value
  if (!node) {
    return null
  }
  return normalizeNodeData(node.data, node.id, node.type ?? 'typed')
})
const selectedRun = computed(() => runs.value.find((run) => run.id === selectedRunId.value) ?? null)
const stepRuntimeByKey = computed<Record<string, CanvasStepRuntime>>(() => {
  const out: Record<string, CanvasStepRuntime> = {}
  for (const item of steps.value) {
    out[item.stepKey] = {
      status: item.status,
      durationMs: typeof item.durationMs === 'number' ? item.durationMs : undefined,
      logRef: typeof item.logRef === 'string' && item.logRef.trim().length > 0 ? item.logRef : undefined,
      artifactCount: countArtifacts(item.artifacts),
      errorCode:
        typeof item.error?.code === 'string' && item.error.code.trim().length > 0 ? item.error.code.trim() : undefined,
    }
  }
  return out
})
const selectedNodeRuntime = computed<CanvasStepRuntime | null>(() => {
  if (!selectedNodeId.value) {
    return null
  }
  return stepRuntimeByKey.value[selectedNodeId.value] ?? null
})

provide(canvasStepRuntimeByKeyKey, stepRuntimeByKey)

const patchDiff = computed(() => {
  const beforeNodes = new Map(baseGraph.value.nodes.map((node) => [node.id, JSON.stringify(node)]))
  const afterNodes = new Map(graphNodes.value.map((node) => [node.id, JSON.stringify(node)]))
  const beforeEdges = new Set(baseGraph.value.edges.map((edge) => edgeKey(edge)))
  const afterEdges = new Set(graphEdges.value.map((edge) => edgeKey(edge)))

  const addedNodes = [...afterNodes.keys()].filter((id) => !beforeNodes.has(id))
  const removedNodes = [...beforeNodes.keys()].filter((id) => !afterNodes.has(id))
  const changedNodes = [...afterNodes.keys()].filter((id) => beforeNodes.has(id) && beforeNodes.get(id) !== afterNodes.get(id))
  const addedEdges = [...afterEdges].filter((id) => !beforeEdges.has(id))
  const removedEdges = [...beforeEdges].filter((id) => !afterEdges.has(id))

  return { addedNodes, removedNodes, changedNodes, addedEdges, removedEdges }
})

watch([graphNodes, graphEdges], () => {
  if (!restoringHistory.value) {
    recordHistory()
  }
}, { deep: true })

watch(selectedRunId, (runId) => {
  stopRunRuntimePolling()
  if (!runId) {
    steps.value = []
    return
  }
  void refreshSelectedRunRuntime(runId)
})

watch(selectedRun, (run) => {
  stopRunRuntimePolling()
  if (useMock || !run || isRunTerminal(run.status)) {
    return
  }
  scheduleRunRuntimePoll(run.id)
})

watch(templates, (items) => {
  if (!items.some((item) => item.id === selectedTemplateId.value)) {
    selectedTemplateId.value = items[0]?.id ?? null
  }
})

watch(selectedTemplateId, (templateId) => {
  if (!templateId) {
    return
  }
  void loadTemplateGraph(templateId)
})

onMounted(() => {
  void onRefreshAll()
})

onBeforeUnmount(() => {
  stopRunRuntimePolling()
})

async function onRefreshAll(): Promise<void> {
  apiError.value = null
  await Promise.all([loadTemplates(), loadRuns()])
  if (selectedRunId.value) {
    await refreshSelectedRunRuntime(selectedRunId.value)
  }
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

async function refreshSelectedRunRuntime(runId: string): Promise<void> {
  if (useMock || selectedRunId.value !== runId) {
    return
  }
  await Promise.all([loadRunByID(runId), loadSteps(runId)])
  const run = runs.value.find((item) => item.id === runId)
  if (!run || isRunTerminal(run.status)) {
    stopRunRuntimePolling()
    return
  }
  scheduleRunRuntimePoll(run.id)
}

async function loadRunByID(runId: string): Promise<void> {
  runsLoading.value = true
  try {
    const run = await getWorkflowRun(runId)
    const list = runs.value.filter((item) => item.id !== run.id)
    runs.value = [run, ...list]
  } catch (error) {
    apiError.value = asApiError(error)
  } finally {
    runsLoading.value = false
  }
}

function scheduleRunRuntimePoll(runId: string): void {
  if (runRuntimeTimer.value || selectedRunId.value !== runId) {
    return
  }
  runRuntimeTimer.value = setTimeout(() => {
    runRuntimeTimer.value = null
    void pollRunRuntime(runId)
  }, 1500)
}

async function pollRunRuntime(runId: string): Promise<void> {
  if (runRuntimePolling.value || selectedRunId.value !== runId || useMock) {
    return
  }
  runRuntimePolling.value = true
  try {
    await refreshSelectedRunRuntime(runId)
  } finally {
    runRuntimePolling.value = false
  }
}

function stopRunRuntimePolling(): void {
  if (!runRuntimeTimer.value) {
    return
  }
  clearTimeout(runRuntimeTimer.value)
  runRuntimeTimer.value = null
}

async function loadTemplateGraph(templateId: string): Promise<void> {
  try {
    if (useMock) {
      graphNodes.value = []
      graphEdges.value = []
      baseGraph.value = { nodes: [], edges: [] }
      resetHistory()
      return
    }
    const template = await getWorkflowTemplate(templateId)
    const graph = normalizeGraphFromTemplate(template.graph)
    graphNodes.value = graph.nodes
    graphEdges.value = graph.edges
    selectedNodeId.value = graph.nodes[0]?.id ?? null
    baseGraph.value = cloneGraph(graph)
    resetHistory()
  } catch (error) {
    apiError.value = asApiError(error)
    graphNodes.value = []
    graphEdges.value = []
    baseGraph.value = { nodes: [], edges: [] }
    resetHistory()
  }
}

function onSelectTemplate(templateId: string): void {
  selectedTemplateId.value = templateId
}

function onSelectRun(runId: string): void {
  selectedRunId.value = runId
}

function onNodeClick(event: unknown): void {
  const payload = event as { node?: Node<GraphNodeData> }
  selectedNodeId.value = payload.node?.id ?? null
}

function onSelectStep(stepKey: string): void {
  if (stepKey.trim().length === 0) {
    return
  }
  selectedNodeId.value = stepKey
}

function onConnect(connection: Connection): void {
  connectionError.value = ''
  if (!connection.source || !connection.target) {
    return
  }
  const sourceNode = graphNodes.value.find((node) => node.id === connection.source)
  const targetNode = graphNodes.value.find((node) => node.id === connection.target)
  const sourceType = sourceNode?.data?.outputType ?? 'any'
  const targetType = targetNode?.data?.inputType ?? 'any'
  if (!isTypeCompatible(sourceType, targetType)) {
    connectionError.value = t('page.canvas.connectionMismatch', { sourceType, targetType })
    pushToast({
      tone: 'error',
      title: t('error.workflow.invalid_request'),
      message: connectionError.value,
    })
    return
  }
  const edge: Edge = {
    id: `e_${connection.source}_${connection.target}_${Date.now()}`,
    source: connection.source,
    target: connection.target,
    sourceHandle: connection.sourceHandle ?? undefined,
    targetHandle: connection.targetHandle ?? undefined,
  }
  graphEdges.value = [...graphEdges.value, edge]
}

function onAddNode(preset: NodePreset): void {
  const id = `node_${Date.now()}_${Math.floor(Math.random() * 1000)}`
  const index = graphNodes.value.length
  const node: Node<GraphNodeData> = {
    id,
    type: 'typed',
    position: { x: 120 + (index % 3) * 220, y: 80 + Math.floor(index / 3) * 120 },
    data: {
      label: preset.label,
      inputType: preset.inputType,
      outputType: preset.outputType,
      nodeType: preset.nodeType,
    },
  }
  graphNodes.value = [...graphNodes.value, node]
  selectedNodeId.value = id
}

function onRemoveSelectedNode(): void {
  if (!selectedNodeId.value) {
    return
  }
  const target = selectedNodeId.value
  graphNodes.value = graphNodes.value.filter((node) => node.id !== target)
  graphEdges.value = graphEdges.value.filter((edge) => edge.source !== target && edge.target !== target)
  selectedNodeId.value = null
}

async function onCreateTemplate(): Promise<void> {
  if (isBusy.value || useMock) {
    return
  }
  const name = templateName.value.trim()
  if (name.length === 0) {
    return
  }
  isBusy.value = true
  try {
    const response = await createWorkflowTemplate({
      name,
      graph: toWorkflowGraph(graphNodes.value, graphEdges.value),
      schemaInputs: {},
      schemaOutputs: {},
      visibility: 'PRIVATE',
    })
    templateName.value = ''
    await loadTemplates()
    selectedTemplateId.value = response.resource.id
    pushToast({
      tone: 'success',
      title: t('page.canvas.actionCreateTemplate'),
      message: response.resource.id,
    })
  } catch (error) {
    const apiErr = asApiError(error)
    apiError.value = apiErr
    pushToast({
      tone: 'error',
      title: t('error.workflow.invalid_request'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    })
  } finally {
    isBusy.value = false
  }
}

async function onPatchTemplate(): Promise<void> {
  if (!selectedTemplateId.value || isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    await patchWorkflowTemplate(selectedTemplateId.value, {
      graph: toWorkflowGraph(graphNodes.value, graphEdges.value),
    })
    baseGraph.value = cloneGraph({ nodes: graphNodes.value, edges: graphEdges.value })
    resetHistory()
    pushToast({
      tone: 'success',
      title: t('page.canvas.actionPatchTemplate'),
      message: selectedTemplateId.value,
    })
  } catch (error) {
    const apiErr = asApiError(error)
    apiError.value = apiErr
    pushToast({
      tone: 'error',
      title: t('error.workflow.invalid_request'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    })
  } finally {
    isBusy.value = false
  }
}

async function onPublishTemplate(): Promise<void> {
  if (!selectedTemplateId.value || isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    await publishWorkflowTemplate(selectedTemplateId.value)
    await loadTemplates()
    pushToast({
      tone: 'success',
      title: t('page.canvas.actionPublishTemplate'),
      message: selectedTemplateId.value,
    })
  } catch (error) {
    const apiErr = asApiError(error)
    apiError.value = apiErr
    pushToast({
      tone: 'error',
      title: t('error.workflow.invalid_request'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    })
  } finally {
    isBusy.value = false
  }
}

async function onRunTemplate(): Promise<void> {
  await submitRun({})
}

async function onRunFromHere(): Promise<void> {
  if (!selectedNodeId.value) {
    return
  }
  await submitRun({ fromStepKey: selectedNodeId.value })
}

async function onTestNode(): Promise<void> {
  if (!selectedNodeId.value) {
    return
  }
  await submitRun({ fromStepKey: selectedNodeId.value, testNode: true })
}

async function submitRun(params: { fromStepKey?: string; testNode?: boolean }): Promise<void> {
  if (!selectedTemplateId.value || isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    const response = await createWorkflowRun({
      templateId: selectedTemplateId.value,
      mode: 'sync',
      inputs: {},
      fromStepKey: params.fromStepKey,
      testNode: params.testNode,
    })
    selectedRunId.value = response.resource.id
    await Promise.all([loadRuns(), loadSteps(response.resource.id)])
    void refreshSelectedRunRuntime(response.resource.id)
    pushToast({
      tone: 'success',
      title: t('page.canvas.actionRunTemplate'),
      message: response.resource.id,
    })
  } catch (error) {
    const apiErr = asApiError(error)
    apiError.value = apiErr
    pushToast({
      tone: 'error',
      title: t('error.workflow.invalid_request'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    })
  } finally {
    isBusy.value = false
  }
}

async function onCancelRun(): Promise<void> {
  if (!selectedRunId.value || isBusy.value || useMock) {
    return
  }
  isBusy.value = true
  try {
    await cancelWorkflowRun(selectedRunId.value)
    await Promise.all([loadRuns(), loadSteps(selectedRunId.value)])
    void refreshSelectedRunRuntime(selectedRunId.value)
    pushToast({
      tone: 'success',
      title: t('page.canvas.actionCancelRun'),
      message: selectedRunId.value,
    })
  } catch (error) {
    const apiErr = asApiError(error)
    apiError.value = apiErr
    pushToast({
      tone: 'error',
      title: t('error.workflow.invalid_request'),
      message: t(apiErr.messageKey || 'error.common.internal', apiErr.details ?? {}),
    })
  } finally {
    isBusy.value = false
  }
}

function onUndo(): void {
  if (!canUndo.value) {
    return
  }
  restoreSnapshot(historyIndex.value - 1)
}

function onRedo(): void {
  if (!canRedo.value) {
    return
  }
  restoreSnapshot(historyIndex.value + 1)
}

function resetHistory(): void {
  historyStack.value = [snapshotGraph(graphNodes.value, graphEdges.value)]
  historyIndex.value = 0
}

function recordHistory(): void {
  const snapshot = snapshotGraph(graphNodes.value, graphEdges.value)
  if (historyStack.value[historyIndex.value] === snapshot) {
    return
  }
  historyStack.value = historyStack.value.slice(0, historyIndex.value + 1)
  historyStack.value.push(snapshot)
  historyIndex.value = historyStack.value.length - 1
}

function restoreSnapshot(nextIndex: number): void {
  const snapshot = historyStack.value[nextIndex]
  if (!snapshot) {
    return
  }
  restoringHistory.value = true
  const parsed = parseSnapshot(snapshot)
  graphNodes.value = parsed.nodes
  graphEdges.value = parsed.edges
  historyIndex.value = nextIndex
  restoringHistory.value = false
}

function snapshotGraph(nodes: Node<GraphNodeData>[], edges: Edge[]): string {
  return JSON.stringify({
    nodes: nodes.map((node) => ({
      id: node.id,
      type: node.type,
      position: node.position,
      data: node.data,
    })),
    edges: edges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      sourceHandle: edge.sourceHandle,
      targetHandle: edge.targetHandle,
    })),
  })
}

function parseSnapshot(raw: string): GraphSnapshot {
  try {
    const parsed = JSON.parse(raw) as { nodes?: any[]; edges?: any[] }
    return normalizeGraphFromTemplate({ nodes: parsed.nodes ?? [], edges: parsed.edges ?? [] })
  } catch {
    return { nodes: [], edges: [] }
  }
}

function normalizeGraphFromTemplate(graphRaw: unknown): GraphSnapshot {
  const graph = (graphRaw ?? {}) as { nodes?: any[]; edges?: any[] }
  const nodes = (Array.isArray(graph.nodes) ? graph.nodes : []).map((raw, index) => normalizeNode(raw, index))
  const edges = (Array.isArray(graph.edges) ? graph.edges : []).map((raw) => normalizeEdge(raw))
  return { nodes, edges }
}

function normalizeNode(raw: any, index: number): Node<GraphNodeData> {
  const nodeType = typeof raw?.type === 'string' ? raw.type : 'noop'
  const fallbackLabel = typeof raw?.label === 'string' ? raw.label : nodeType
  const nodeData = normalizeNodeData(raw?.data, fallbackLabel, nodeType)
  return {
    id: typeof raw?.id === 'string' && raw.id.trim().length > 0 ? raw.id.trim() : `node_${index + 1}`,
    type: 'typed',
    position: {
      x: typeof raw?.position?.x === 'number' ? raw.position.x : 120 + (index % 3) * 220,
      y: typeof raw?.position?.y === 'number' ? raw.position.y : 80 + Math.floor(index / 3) * 120,
    },
    data: nodeData,
  }
}

function normalizeNodeData(raw: unknown, fallbackLabel: string, fallbackNodeType: string): GraphNodeData {
  const data = (raw ?? {}) as Partial<GraphNodeData>
  const defaults = inferPortTypes(fallbackNodeType)
  return {
    label: typeof data.label === 'string' && data.label.trim().length > 0 ? data.label : fallbackLabel,
    inputType: typeof data.inputType === 'string' && data.inputType.trim().length > 0 ? data.inputType : defaults.inputType,
    outputType: typeof data.outputType === 'string' && data.outputType.trim().length > 0 ? data.outputType : defaults.outputType,
    nodeType:
      typeof data.nodeType === 'string' && data.nodeType.trim().length > 0 ? data.nodeType : fallbackNodeType,
  }
}

function normalizeEdge(raw: any): Edge {
  const source = typeof raw?.source === 'string' ? raw.source : typeof raw?.from === 'string' ? raw.from : ''
  const target = typeof raw?.target === 'string' ? raw.target : typeof raw?.to === 'string' ? raw.to : ''
  return {
    id: typeof raw?.id === 'string' && raw.id.trim().length > 0 ? raw.id : `e_${source}_${target}`,
    source,
    target,
    sourceHandle: typeof raw?.sourceHandle === 'string' ? raw.sourceHandle : undefined,
    targetHandle: typeof raw?.targetHandle === 'string' ? raw.targetHandle : undefined,
  }
}

function inferPortTypes(nodeType: string): { inputType: string; outputType: string } {
  if (nodeType.includes('source')) {
    return { inputType: 'none', outputType: 'text' }
  }
  if (nodeType.includes('transform')) {
    return { inputType: 'text', outputType: 'json' }
  }
  if (nodeType.includes('sink')) {
    return { inputType: 'json', outputType: 'any' }
  }
  return { inputType: 'any', outputType: 'any' }
}

function toWorkflowGraph(nodes: Node<GraphNodeData>[], edges: Edge[]): Record<string, unknown> {
  return {
    nodes: nodes.map((node) => ({
      id: node.id,
      type: node.data?.nodeType ?? node.type ?? 'noop',
      position: node.position,
      data: {
        label: node.data?.label ?? node.id,
        inputType: node.data?.inputType ?? 'any',
        outputType: node.data?.outputType ?? 'any',
      },
    })),
    edges: edges.map((edge) => ({
      id: edge.id,
      from: edge.source,
      to: edge.target,
      source: edge.source,
      target: edge.target,
      sourceHandle: edge.sourceHandle,
      targetHandle: edge.targetHandle,
    })),
  }
}

function edgeKey(edge: Edge): string {
  return `${edge.source}->${edge.target}`
}

function isTypeCompatible(sourceType: string, targetType: string): boolean {
  if (sourceType === 'any' || targetType === 'any') {
    return true
  }
  if (targetType === 'none') {
    return false
  }
  return sourceType === targetType
}

function isRunTerminal(status: string): boolean {
  return status === 'succeeded' || status === 'failed' || status === 'canceled'
}

function countArtifacts(artifacts: Record<string, unknown> | undefined): number {
  if (!artifacts) {
    return 0
  }
  if (Array.isArray(artifacts.items)) {
    return artifacts.items.length
  }
  if (Array.isArray(artifacts.assets)) {
    return artifacts.assets.length
  }
  return Object.keys(artifacts).length
}

function cloneGraph(graph: GraphSnapshot): GraphSnapshot {
  return {
    nodes: graph.nodes.map((node) => ({
      ...node,
      data: normalizeNodeData(node.data, node.id, node.type ?? 'noop'),
      position: { ...node.position },
    })),
    edges: graph.edges.map((edge) => ({ ...edge })),
  }
}

function asApiError(error: unknown): ApiError {
  if (error instanceof ApiHttpError) {
    return error.error
  }
  return {
    code: 'INTERNAL_ERROR',
    messageKey: 'error.common.internal',
  }
}
</script>
