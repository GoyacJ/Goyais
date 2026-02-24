<template>
  <SettingsShell
    active-key="settings_general"
    :title="t('menu.settingsGeneral')"
    :subtitle="`${t('settings.scope')} / General`"
    runtime-status-mode
    :runtime-conversation-status="workspaceStatus.conversationStatus.value"
    :runtime-connection-status="workspaceStatus.connectionStatus.value"
    :runtime-user-display-name="workspaceStatus.userDisplayName.value"
    :runtime-hub-url="workspaceStatus.hubURL.value"
  >
    <div class="general-scroll">
      <GeneralSettingsSection
        class="settings-section"
        :title="t('settings.general.section.startup.title')"
        :description="t('settings.general.section.startup.description')"
      >
        <GeneralSettingsRow
          :label="t('settings.general.field.launchOnStartup.label')"
          :description="t('settings.general.field.launchOnStartup.description')"
          :unsupported-reason="launchUnsupportedReason"
        >
          <BaseSelect
            v-model="launchOnStartupModel"
            :options="enabledDisabledOptions"
            :disabled="settings.loading.value || !settings.capability.value.launchOnStartup.supported"
          />
        </GeneralSettingsRow>
      </GeneralSettingsSection>

      <GeneralSettingsSection
        class="settings-section"
        :title="t('settings.general.section.directory.title')"
        :description="t('settings.general.section.directory.description')"
      >
        <GeneralSettingsRow
          :label="t('settings.general.field.defaultProjectDirectory.label')"
          :description="t('settings.general.field.defaultProjectDirectory.description')"
          :hint="t('settings.general.field.defaultProjectDirectory.hint')"
        >
          <BaseInput
            v-model="defaultDirectoryModel"
            :disabled="settings.loading.value"
            :placeholder="t('settings.general.field.defaultProjectDirectory.placeholder')"
          />
        </GeneralSettingsRow>
      </GeneralSettingsSection>

      <GeneralSettingsSection
        class="settings-section"
        :title="t('settings.general.section.notifications.title')"
        :description="t('settings.general.section.notifications.description')"
      >
        <GeneralSettingsRow
          :label="t('settings.general.field.notificationsReconnect.label')"
          :description="t('settings.general.field.notificationsReconnect.description')"
          :unsupported-reason="notificationsUnsupportedReason"
        >
          <BaseSelect
            v-model="notificationsReconnectModel"
            :options="enabledDisabledOptions"
            :disabled="settings.loading.value || !settings.capability.value.notifications.supported"
          />
        </GeneralSettingsRow>

        <GeneralSettingsRow
          :label="t('settings.general.field.notificationsApproval.label')"
          :description="t('settings.general.field.notificationsApproval.description')"
          :unsupported-reason="notificationsUnsupportedReason"
        >
          <BaseSelect
            v-model="notificationsApprovalModel"
            :options="enabledDisabledOptions"
            :disabled="settings.loading.value || !settings.capability.value.notifications.supported"
          />
        </GeneralSettingsRow>

        <GeneralSettingsRow
          :label="t('settings.general.field.notificationsError.label')"
          :description="t('settings.general.field.notificationsError.description')"
          :unsupported-reason="notificationsUnsupportedReason"
        >
          <BaseSelect
            v-model="notificationsErrorModel"
            :options="enabledDisabledOptions"
            :disabled="settings.loading.value || !settings.capability.value.notifications.supported"
          />
        </GeneralSettingsRow>
      </GeneralSettingsSection>

      <GeneralSettingsSection
        class="settings-section"
        :title="t('settings.general.section.telemetry.title')"
        :description="t('settings.general.section.telemetry.description')"
      >
        <GeneralSettingsRow
          :label="t('settings.general.field.telemetryLevel.label')"
          :description="t('settings.general.field.telemetryLevel.description')"
        >
          <BaseSelect v-model="telemetryLevelModel" :options="telemetryOptions" :disabled="settings.loading.value" />
        </GeneralSettingsRow>
      </GeneralSettingsSection>

      <GeneralSettingsSection
        class="settings-section"
        :title="t('settings.general.section.updatePolicy.title')"
        :description="t('settings.general.section.updatePolicy.description')"
      >
        <GeneralSettingsRow
          :label="t('settings.general.field.updateChannel.label')"
          :description="t('settings.general.field.updateChannel.description')"
        >
          <BaseSelect v-model="updateChannelModel" :options="updateChannelOptions" :disabled="settings.loading.value" />
        </GeneralSettingsRow>

        <GeneralSettingsRow
          :label="t('settings.general.field.updateCheckFrequency.label')"
          :description="t('settings.general.field.updateCheckFrequency.description')"
        >
          <BaseSelect v-model="updateFrequencyModel" :options="updateFrequencyOptions" :disabled="settings.loading.value" />
        </GeneralSettingsRow>

        <GeneralSettingsRow
          :label="t('settings.general.field.updateAutoDownload.label')"
          :description="t('settings.general.field.updateAutoDownload.description')"
        >
          <BaseSelect
            v-model="updateAutoDownloadModel"
            :options="enabledDisabledOptions"
            :disabled="settings.loading.value"
          />
        </GeneralSettingsRow>

        <GeneralSettingsRow
          :label="t('settings.general.field.currentVersion.label')"
          :description="t('settings.general.field.currentVersion.description')"
        >
          <p class="version-value">{{ currentVersionText }}</p>
        </GeneralSettingsRow>

        <GeneralSettingsRow
          :label="t('settings.general.field.checkVersion.label')"
          :description="t('settings.general.field.checkVersion.description')"
          :hint="checkVersionHint"
          :unsupported-reason="checkVersionUnsupportedReason"
        >
          <BaseButton variant="secondary" :disabled="checkVersionButtonDisabled" @click="checkVersion">
            {{ checkVersionActionLabel }}
          </BaseButton>
        </GeneralSettingsRow>
      </GeneralSettingsSection>

      <GeneralSettingsSection
        class="settings-section"
        :title="t('settings.general.section.diagnostics.title')"
        :description="t('settings.general.section.diagnostics.description')"
      >
        <GeneralSettingsRow
          :label="t('settings.general.field.diagnosticsLevel.label')"
          :description="t('settings.general.field.diagnosticsLevel.description')"
        >
          <BaseSelect
            v-model="diagnosticsLevelModel"
            :options="diagnosticsLevelOptions"
            :disabled="settings.loading.value"
          />
        </GeneralSettingsRow>

        <GeneralSettingsRow
          :label="t('settings.general.field.logRetentionDays.label')"
          :description="t('settings.general.field.logRetentionDays.description')"
        >
          <BaseSelect v-model="logRetentionModel" :options="logRetentionOptions" :disabled="settings.loading.value" />
        </GeneralSettingsRow>
      </GeneralSettingsSection>

      <section class="footer-actions">
        <p class="status" v-if="settings.error.value !== ''">{{ settings.error.value }}</p>
        <p class="status" v-else-if="settings.saving.value">{{ t("settings.general.status.saving") }}</p>
        <p class="status" v-else>{{ t("settings.general.status.ready") }}</p>

        <BaseButton variant="danger" :disabled="settings.loading.value || settings.saving.value" @click="resetAll">
          {{ t("settings.general.reset.action") }}
        </BaseButton>
      </section>
    </div>
  </SettingsShell>
</template>

<script setup lang="ts">
import { computed } from "vue";

import { projectStore } from "@/modules/project/store";
import GeneralSettingsRow from "@/modules/workspace/components/general/GeneralSettingsRow.vue";
import GeneralSettingsSection from "@/modules/workspace/components/general/GeneralSettingsSection.vue";
import { useSettingsGeneralViewModel } from "@/modules/workspace/views/useSettingsGeneralViewModel";
import SettingsShell from "@/shared/shells/SettingsShell.vue";
import { useWorkspaceStatusSync } from "@/shared/stores/workspaceStatusStore";
import BaseButton from "@/shared/ui/BaseButton.vue";
import BaseInput from "@/shared/ui/BaseInput.vue";
import BaseSelect from "@/shared/ui/BaseSelect.vue";

const {
  t,
  settings,
  enabledDisabledOptions,
  telemetryOptions,
  updateChannelOptions,
  updateFrequencyOptions,
  diagnosticsLevelOptions,
  logRetentionOptions,
  launchOnStartupModel,
  defaultDirectoryModel,
  notificationsReconnectModel,
  notificationsApprovalModel,
  notificationsErrorModel,
  telemetryLevelModel,
  updateChannelModel,
  updateFrequencyModel,
  updateAutoDownloadModel,
  diagnosticsLevelModel,
  logRetentionModel,
  launchUnsupportedReason,
  notificationsUnsupportedReason,
  currentVersionText,
  checkVersionUnsupportedReason,
  checkVersionButtonDisabled,
  checkVersionActionLabel,
  checkVersionHint,
  checkVersion,
  resetAll
} = useSettingsGeneralViewModel();

const workspaceStatus = useWorkspaceStatusSync({
  conversationId: computed(() => projectStore.activeConversationId)
});
</script>

<style scoped>
.settings-section {
  margin-bottom: 0;
}

.general-scroll {
  min-height: 0;
  max-height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  display: grid;
  gap: var(--global-space-12);
  padding-right: var(--global-space-4);
}

.footer-actions {
  border: 1px dashed var(--semantic-border);
  border-radius: var(--global-radius-12);
  background: var(--semantic-surface);
  padding: var(--global-space-12);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--global-space-12);
}

.status {
  margin: 0;
  color: var(--semantic-text-muted);
  font-size: var(--global-font-size-12);
}

.version-value {
  margin: 0;
  font-size: var(--global-font-size-12);
  color: var(--semantic-text);
}

@media (max-width: 960px) {
  .footer-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
