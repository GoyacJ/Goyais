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
          <LogPanel v-else :lines="eventLines" />
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
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n({ useScope: 'global' })
const { pushToast } = useToast()

const sessions = ref<AISessionDTO[]>([])
const selectedSessionId = ref('')
const selectedSession = ref<AISessionDTO | null>(null)
const eventLines = ref<string[]>([])

const createTitle = ref('')
const createGoal = ref('')
const turnMessage = ref('')
const turnMode = ref<'plan' | 'execute'>('plan')

const isRefreshing = ref(false)
const isSubmitting = ref(false)

const windowPanes = computed(() => [
  { id: 'ai-sessions', title: t('page.ai.sessionsTitle') },
  { id: 'ai-composer', title: t('page.ai.composerTitle') },
  { id: 'ai-events', title: t('page.ai.eventsTitle') },
])

const turnModeOptions = computed(() => [
  { value: 'plan', label: t('page.ai.turnModePlan') },
  { value: 'execute', label: t('page.ai.turnModeExecute') },
])

const canSendTurn = computed(
  () =>
    selectedSession.value?.status === 'active' &&
    turnMessage.value.trim().length > 0 &&
    !isSubmitting.value,
)

onMounted(() => {
  void loadSessions()
})

watch(
  selectedSessionId,
  () => {
    void loadSelectedSessionContext()
  },
  { immediate: true },
)

async function loadSessions(): Promise<void> {
  isRefreshing.value = true
  try {
    const response = await listAISessions({ page: 1, pageSize: 200 })
    sessions.value = response.items

    if (sessions.value.length === 0) {
      selectedSessionId.value = ''
      selectedSession.value = null
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
    eventLines.value = []
    return
  }
  try {
    const [session, events] = await Promise.all([getAISession(sessionID), getAISessionEvents(sessionID)])
    selectedSession.value = session
    eventLines.value = events.map(formatEventLine)
  } catch (error) {
    notifyError(t('common.refresh'), error)
  }
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
    const response = await createAISessionTurn(sessionID, {
      message,
      execute: turnMode.value === 'execute',
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

function notifyError(action: string, error: unknown): void {
  const apiError = toApiError(error)
  pushToast({
    title: action,
    message: t(apiError.messageKey || 'error.common.internal', apiError.details ?? {}),
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
</script>
