<template>
  <section class="ui-page">
    <PageHeader :title="t('page.ai.title')" :subtitle="t('page.ai.subtitle')">
      <template #actions>
        <Button variant="secondary" :disabled="isRefreshing" @click="onRefresh">
          <Icon name="refresh" :size="14" decorative />
          {{ t('common.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <WindowBoard route-key="ai-workbench" :panes="windowPanes">
      <template #ai-sessions>
        <SectionCard :title="t('page.ai.sessionsTitle')" :subtitle="t('page.ai.sessionsSubtitle')">
          <div class="space-y-3">
            <div class="space-y-2 rounded-card border border-ui-border bg-ui-surface2 p-3">
              <Input v-model="createTitle" :placeholder="t('page.ai.createTitlePlaceholder')" />
              <Input v-model="createGoal" :placeholder="t('page.ai.createGoalPlaceholder')" />
              <Button class="w-full" :disabled="isSubmitting" @click="onCreateSession">
                {{ t('page.ai.actionCreateSession') }}
              </Button>
            </div>

            <p v-if="isRefreshing" class="text-xs text-ui-muted">{{ t('common.loading') }}</p>
            <EmptyState
              v-else-if="sessions.length === 0"
              variant="commands-empty"
              :title="t('page.ai.emptySessionsTitle')"
              :description="t('page.ai.emptySessionsDescription')"
            />

            <ul v-else class="space-y-2">
              <li v-for="item in sessions" :key="item.id">
                <button
                  type="button"
                  class="ui-control ui-focus-ring ui-pressable w-full text-left"
                  :class="selectedSessionId === item.id ? 'border-ui-primary bg-ui-primary/10' : ''"
                  @click="selectedSessionId = item.id"
                >
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0">
                      <p class="truncate text-sm font-semibold text-ui-fg">{{ item.title || item.id }}</p>
                      <p class="truncate text-xs text-ui-muted">{{ item.goal || '-' }}</p>
                      <p class="ui-monospace mt-1 truncate text-[11px] text-ui-muted">{{ item.id }}</p>
                    </div>
                    <Badge :tone="sessionTone(item.status)">{{ sessionStatusLabel(item.status) }}</Badge>
                  </div>
                </button>
              </li>
            </ul>
          </div>
        </SectionCard>
      </template>

      <template #ai-composer>
        <SectionCard :title="t('page.ai.composerTitle')" :subtitle="t('page.ai.composerSubtitle')">
          <div v-if="selectedSession" class="space-y-3">
            <div class="rounded-card border border-ui-border bg-ui-surface2 p-3">
              <p class="text-xs text-ui-muted">{{ t('page.ai.selectedSession') }}</p>
              <p class="mt-1 text-sm font-semibold text-ui-fg">{{ selectedSession.title || selectedSession.id }}</p>
              <p class="ui-monospace mt-1 text-[11px] text-ui-muted">{{ selectedSession.id }}</p>
            </div>

            <Textarea
              v-model="turnMessage"
              :rows="8"
              :placeholder="t('page.ai.turnPlaceholder')"
            />
            <div class="rounded-card border border-ui-border bg-ui-surface2 p-3">
              <p class="text-xs font-semibold text-ui-fg">{{ t('page.ai.planPreviewTitle') }}</p>
              <template v-if="turnPlanPreview">
                <p class="mt-1 text-xs text-ui-muted">
                  {{ t('page.ai.planPreviewCommand') }}: {{ turnPlanPreview.commandType }}
                </p>
                <pre class="mt-2 overflow-x-auto rounded border border-ui-border bg-ui-surface px-2 py-1 text-[11px] text-ui-fg">{{
                  formatJSON(turnPlanPreview.payload)
                }}</pre>
              </template>
              <p v-else class="mt-1 text-xs text-ui-muted">{{ t('page.ai.planPreviewEmpty') }}</p>
            </div>
            <div class="grid gap-2 md:grid-cols-[1fr_auto_auto]">
              <Select v-model="turnMode" :options="turnModeOptions" />
              <Button :disabled="!canSendTurn" @click="onSendTurn">
                {{ t('page.ai.actionSendTurn') }}
              </Button>
              <Button
                variant="secondary"
                :disabled="selectedSession.status !== 'active' || isSubmitting"
                @click="onArchiveSession"
              >
                {{ t('page.ai.actionArchiveSession') }}
              </Button>
            </div>
          </div>

          <EmptyState
            v-else
            variant="commands-empty"
            :title="t('page.ai.emptySessionsTitle')"
            :description="t('page.ai.emptySessionsDescription')"
          />
        </SectionCard>
      </template>

      <template #ai-events>
        <SectionCard :title="t('page.ai.eventsTitle')" :subtitle="t('page.ai.eventsSubtitle')">
          <EmptyState
            v-if="eventLines.length === 0"
            variant="commands-empty"
            :title="t('page.ai.emptyEventsTitle')"
            :description="t('page.ai.emptyEventsDescription')"
          />
          <div v-else class="space-y-3">
            <div v-if="feedbackTimeline.length > 0" class="space-y-2">
              <div
                v-for="item in feedbackTimeline"
                :key="item.id"
                class="rounded-card border border-ui-border bg-ui-surface2 px-3 py-2"
              >
                <p class="text-xs font-semibold text-ui-fg">{{ item.title }}</p>
                <p class="mt-1 text-xs text-ui-muted">{{ item.detail }}</p>
                <p v-if="item.timestamp" class="mt-1 text-[11px] text-ui-muted">{{ item.timestamp }}</p>
              </div>
            </div>
            <LogPanel :lines="eventLines" />
          </div>
        </SectionCard>
      </template>
    </WindowBoard>
  </section>
</template>

<script setup lang="ts">
import {
  archiveAISession,
  createAISession,
  createAISessionTurn,
  getAISession,
  getAISessionEvents,
  type AISessionEvent,
  listAISessions,
} from '@/api/ai'
import { ApiHttpError } from '@/api/http'
import type { AISessionDTO, ApiError } from '@/api/types'
import EmptyState from '@/components/layout/EmptyState.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import SectionCard from '@/components/layout/SectionCard.vue'
import WindowBoard from '@/components/layout/WindowBoard.vue'
import LogPanel from '@/components/runtime/LogPanel.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Textarea from '@/components/ui/Textarea.vue'
import { useToast } from '@/composables/useToast'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const sessions = ref<AISessionDTO[]>([])
const selectedSessionId = ref('')
const selectedSession = ref<AISessionDTO | null>(null)
const sessionEvents = ref<AISessionEvent[]>([])
const eventLines = ref<string[]>([])

const createTitle = ref('')
const createGoal = ref('')
const turnMessage = ref('')
const turnMode = ref<'plan' | 'execute'>('plan')

const isRefreshing = ref(false)
const isSubmitting = ref(false)
const eventsPollTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const eventsPollingInFlight = ref(false)

const windowPanes = computed(() => [
  { id: 'ai-sessions', title: t('page.ai.sessionsTitle') },
  { id: 'ai-composer', title: t('page.ai.composerTitle') },
  { id: 'ai-events', title: t('page.ai.eventsTitle') },
])

const turnModeOptions = computed(() => [
  { value: 'plan', label: t('page.ai.turnModePlan') },
  { value: 'execute', label: t('page.ai.turnModeExecute') },
])

type TurnPlanPreview = {
  commandType: string
  payload: Record<string, unknown>
}

type TimelineItem = {
  id: string
  title: string
  detail: string
  timestamp: string
}

const turnPlanPreview = computed<TurnPlanPreview | null>(() => {
  return inferTurnPlan(turnMessage.value.trim())
})

const feedbackTimeline = computed<TimelineItem[]>(() => {
  return sessionEvents.value.map(buildTimelineItem).filter((item): item is TimelineItem => item !== null)
})

const canSendTurn = computed(
  () =>
    selectedSession.value?.status === 'active' &&
    turnMessage.value.trim().length > 0 &&
    !isSubmitting.value,
)

onMounted(() => {
  void loadSessions()
})

onBeforeUnmount(() => {
  stopEventsPolling()
})

watch(
  selectedSessionId,
  () => {
    stopEventsPolling()
    void loadSelectedSessionContext()
  },
  { immediate: true },
)

watch(
  selectedSession,
  (session) => {
    stopEventsPolling()
    if (!session || session.status !== 'active') {
      return
    }
    scheduleEventsPolling(session.id)
  },
  { deep: false },
)

async function loadSessions(): Promise<void> {
  isRefreshing.value = true
  try {
    const response = await listAISessions({ page: 1, pageSize: 200 })
    sessions.value = response.items

    if (sessions.value.length === 0) {
      selectedSessionId.value = ''
      selectedSession.value = null
      sessionEvents.value = []
      eventLines.value = []
      return
    }

    const hasSelected = sessions.value.some((item) => item.id === selectedSessionId.value)
    if (!hasSelected) {
      selectedSessionId.value = sessions.value[0]?.id ?? ''
    }
  } catch (error) {
    notifyError(t('common.refresh'), error)
  } finally {
    isRefreshing.value = false
  }
}

async function loadSelectedSessionContext(): Promise<void> {
  const sessionID = selectedSessionId.value.trim()
  if (sessionID.length === 0) {
    selectedSession.value = null
    sessionEvents.value = []
    eventLines.value = []
    return
  }
  try {
    const [session, events] = await Promise.all([getAISession(sessionID), getAISessionEvents(sessionID)])
    selectedSession.value = session
    sessionEvents.value = events
    eventLines.value = events.map(formatEventLine)
  } catch (error) {
    notifyError(t('common.refresh'), error)
  }
}

async function refreshSelectedSessionEvents(sessionID: string): Promise<void> {
  if (sessionID.trim().length === 0 || selectedSessionId.value !== sessionID) {
    return
  }
  const events = await getAISessionEvents(sessionID)
  if (selectedSessionId.value !== sessionID) {
    return
  }
  sessionEvents.value = events
  eventLines.value = events.map(formatEventLine)
}

function scheduleEventsPolling(sessionID: string): void {
  if (eventsPollTimer.value || selectedSessionId.value !== sessionID) {
    return
  }
  eventsPollTimer.value = setTimeout(() => {
    eventsPollTimer.value = null
    void pollSelectedSessionEvents(sessionID)
  }, 2000)
}

async function pollSelectedSessionEvents(sessionID: string): Promise<void> {
  if (
    eventsPollingInFlight.value ||
    selectedSessionId.value !== sessionID ||
    selectedSession.value?.status !== 'active' ||
    isSubmitting.value
  ) {
    if (selectedSessionId.value === sessionID && selectedSession.value?.status === 'active') {
      scheduleEventsPolling(sessionID)
    }
    return
  }
  eventsPollingInFlight.value = true
  try {
    await refreshSelectedSessionEvents(sessionID)
  } catch {
    // Keep polling resilient to transient read errors.
  } finally {
    eventsPollingInFlight.value = false
  }
  if (selectedSessionId.value === sessionID && selectedSession.value?.status === 'active') {
    scheduleEventsPolling(sessionID)
  }
}

function stopEventsPolling(): void {
  if (!eventsPollTimer.value) {
    return
  }
  clearTimeout(eventsPollTimer.value)
  eventsPollTimer.value = null
}

async function onCreateSession(): Promise<void> {
  if (isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  try {
    const response = await createAISession({
      title: createTitle.value.trim(),
      goal: createGoal.value.trim(),
      visibility: 'PRIVATE',
    })

    createTitle.value = ''
    createGoal.value = ''
    selectedSessionId.value = response.resource.id
    await loadSessions()
    await loadSelectedSessionContext()

    pushToast({
      title: t('page.ai.actionCreateSession'),
      message: `commandId: ${response.commandRef.commandId}`,
      tone: 'success',
    })
  } catch (error) {
    notifyError(t('page.ai.actionCreateSession'), error)
  } finally {
    isSubmitting.value = false
  }
}

async function onSendTurn(): Promise<void> {
  const sessionID = selectedSessionId.value.trim()
  const message = turnMessage.value.trim()
  if (sessionID.length === 0 || message.length === 0 || isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  try {
    const preview = inferTurnPlan(message)
    const response = await createAISessionTurn(sessionID, {
      message,
      execute: turnMode.value === 'execute',
      intentCommandType: preview?.commandType,
      intentPayload: preview?.payload,
    })
    turnMessage.value = ''
    await loadSessions()
    await loadSelectedSessionContext()

    pushToast({
      title: t('page.ai.actionSendTurn'),
      message: `commandId: ${response.commandRef.commandId}`,
      tone: 'success',
    })
  } catch (error) {
    notifyError(t('page.ai.actionSendTurn'), error)
  } finally {
    isSubmitting.value = false
  }
}

async function onArchiveSession(): Promise<void> {
  const sessionID = selectedSessionId.value.trim()
  if (sessionID.length === 0 || isSubmitting.value) {
    return
  }

  isSubmitting.value = true
  try {
    const response = await archiveAISession(sessionID)
    await loadSessions()
    await loadSelectedSessionContext()

    pushToast({
      title: t('page.ai.actionArchiveSession'),
      message: `commandId: ${response.commandRef.commandId}`,
      tone: 'success',
    })
  } catch (error) {
    notifyError(t('page.ai.actionArchiveSession'), error)
  } finally {
    isSubmitting.value = false
  }
}

async function onRefresh(): Promise<void> {
  await loadSessions()
  await loadSelectedSessionContext()
}

function sessionTone(status: string): 'primary' | 'neutral' {
  return status === 'active' ? 'primary' : 'neutral'
}

function sessionStatusLabel(status: string): string {
  if (status === 'active') {
    return t('page.ai.statusActive')
  }
  if (status === 'archived') {
    return t('page.ai.statusArchived')
  }
  return status
}

function formatEventLine(item: AISessionEvent): string {
  const prefix = item.event ?? 'message'
  if (typeof item.data === 'string') {
    return `${prefix}: ${item.data}`
  }
  if (!item.data || typeof item.data !== 'object') {
    return prefix
  }

  const eventPayload = item.data as Record<string, unknown>
  if (prefix.startsWith('command.')) {
    const commandType = readString(eventPayload, 'commandType')
    const commandID = readString(eventPayload, 'commandId') || readString(eventPayload, 'id')
    const status = readString(eventPayload, 'status')
    const errorCode = readString(eventPayload, 'errorCode')
    const messageKey = readString(eventPayload, 'messageKey')
    const suffix = errorCode || messageKey ? ` (${errorCode || ''}${errorCode && messageKey ? ' / ' : ''}${messageKey || ''})` : ''
    return `${prefix}: ${commandType || 'command'} ${commandID} ${status}${suffix}`.trim()
  }
  if (prefix.startsWith('workflow.')) {
    const runID = readString(eventPayload, 'runId')
    const status = readString(eventPayload, 'status')
    return `${prefix}: run=${runID} status=${status}`.trim()
  }

  const role = readString(eventPayload, 'role')
  const content = readString(eventPayload, 'content')
  const createdAt = readString(eventPayload, 'createdAt')
  if (role || content) {
    const createdSuffix = createdAt ? ` (${createdAt})` : ''
    return `${prefix}: [${role || 'event'}] ${content || ''}${createdSuffix}`.trim()
  }
  return `${prefix}: ${JSON.stringify(eventPayload)}`
}

function readString(payload: Record<string, unknown>, key: string): string {
  const raw = payload[key]
  return typeof raw === 'string' ? raw : ''
}

function inferTurnPlan(message: string): TurnPlanPreview | null {
  const tokens = message.split(/\s+/).filter((item) => item.length > 0)
  if (tokens.length === 0) {
    return null
  }

  if (tokens.length >= 3 && eqToken(tokens[0], 'run') && eqToken(tokens[1], 'workflow')) {
    return workflowRunPlan(cleanToken(tokens[2]))
  }
  if (tokens.length >= 2 && eqToken(tokens[0], 'workflow.run')) {
    return workflowRunPlan(cleanToken(tokens[1]))
  }
  if (tokens.length >= 3 && eqToken(tokens[0], 'retry') && eqToken(tokens[1], 'workflow')) {
    return workflowRetryPlan(cleanToken(tokens[2]))
  }
  if (tokens.length >= 2 && eqToken(tokens[0], 'workflow.retry')) {
    return workflowRetryPlan(cleanToken(tokens[1]))
  }
  if (tokens.length >= 3 && eqToken(tokens[0], 'cancel') && eqToken(tokens[1], 'workflow')) {
    return workflowCancelPlan(cleanToken(tokens[2]))
  }
  if (tokens.length >= 2 && eqToken(tokens[0], 'workflow.cancel')) {
    return workflowCancelPlan(cleanToken(tokens[1]))
  }
  return null
}

function workflowRunPlan(templateId: string): TurnPlanPreview | null {
  if (!templateId) {
    return null
  }
  return {
    commandType: 'workflow.run',
    payload: {
      templateId,
      mode: 'sync',
      inputs: {},
    },
  }
}

function workflowRetryPlan(runId: string): TurnPlanPreview | null {
  if (!runId) {
    return null
  }
  return {
    commandType: 'workflow.retry',
    payload: {
      runId,
      mode: 'sync',
    },
  }
}

function workflowCancelPlan(runId: string): TurnPlanPreview | null {
  if (!runId) {
    return null
  }
  return {
    commandType: 'workflow.cancel',
    payload: {
      runId,
    },
  }
}

function eqToken(left: string, right: string): boolean {
  return left.toLowerCase() === right.toLowerCase()
}

function cleanToken(raw: string): string {
  return raw.trim().replace(/^[`"'()[\]{}.,;:]+|[`"'()[\]{}.,;:]+$/g, '')
}

function formatJSON(value: Record<string, unknown>): string {
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return '{}'
  }
}

function buildTimelineItem(item: AISessionEvent): TimelineItem | null {
  const eventType = item.event ?? ''
  if (!item.data || typeof item.data !== 'object') {
    return null
  }
  const payload = item.data as Record<string, unknown>

  if (eventType.startsWith('command.')) {
    const commandType = readString(payload, 'commandType') || 'command'
    const commandID = readString(payload, 'commandId') || readString(payload, 'id')
    const status = readString(payload, 'status')
    const errorCode = readString(payload, 'errorCode')
    const messageKey = readString(payload, 'messageKey')
    const fallbackID = `${eventType}:${commandType}:${status}`
    const detailParts: string[] = []
    if (commandID) {
      detailParts.push(`commandId=${commandID}`)
    } else {
      detailParts.push(eventType)
    }
    if (errorCode) {
      detailParts.push(`errorCode=${errorCode}`)
    }
    if (messageKey) {
      detailParts.push(`messageKey=${messageKey}`)
    }
    return {
      id: `${eventType}:${commandID || fallbackID}`,
      title: `${commandType} · ${status || eventType}`,
      detail: detailParts.join(' · '),
      timestamp: readString(payload, 'updatedAt') || readString(payload, 'acceptedAt') || readString(payload, 'finishedAt'),
    }
  }
  if (eventType.startsWith('workflow.')) {
    const runID = readString(payload, 'runId')
    const status = readString(payload, 'status')
    const fallbackID = `${eventType}:${status}`
    return {
      id: `${eventType}:${runID || fallbackID}`,
      title: `workflow · ${status || eventType}`,
      detail: runID ? `runId=${runID}` : eventType,
      timestamp: '',
    }
  }
  if (eventType.startsWith('ai.turn.')) {
    const role = readString(payload, 'role') || eventType.replace('ai.turn.', '')
    const content = readString(payload, 'content')
    const turnID = readString(payload, 'id') || `${eventType}:${role}:${readString(payload, 'createdAt')}`
    return {
      id: `${eventType}:${turnID}`,
      title: `ai.turn · ${role}`,
      detail: content || '-',
      timestamp: readString(payload, 'createdAt'),
    }
  }
  return null
}

function notifyError(action: string, error: unknown): void {
  const apiError = toApiError(error)
  const reason = readReason(apiError.details)
  const localized = t(apiError.messageKey || 'error.common.internal', apiError.details ?? {})
  pushToast({
    title: action,
    message: reason ? `${localized} (${reason})` : localized,
    tone: 'error',
  })
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiHttpError) {
    return error.error
  }
  return {
    code: 'INTERNAL_ERROR',
    messageKey: 'error.common.internal',
  }
}

function readReason(details: ApiError['details']): string {
  if (!details || typeof details !== 'object') {
    return ''
  }
  const raw = details.reason
  return typeof raw === 'string' ? raw.trim() : ''
}
</script>
