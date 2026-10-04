<template>
  <!-- Who can sign in, and what Dozzle may do to containers: the wizard's first two
       steps, as the same components. -->
  <SettingsSection :title="$t('setup.steps.login')" :desc="$t('settings.login-desc')">
    <!-- One notice for the page, naming what each section below is set with. -->
    <SetupAccessNotice v-if="status" :status="status" :envs="unlockedEnvs" />
    <SetupLoginForm v-if="status" :status="status" standalone :notices="false" />
    <SetupStatusMissing v-else-if="server" :loading="loading" />
    <div v-else class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <ReadOnlyRow :label="$t('setup.restart.change-auth')" :value="`auth: ${config.authProvider}`" />
    </div>
  </SettingsSection>

  <SettingsSection :title="$t('setup.steps.actions')" :desc="$t('setup.actions.subtitle')">
    <SetupTogglesForm v-if="status" :status="status" :notices="false" autosave />
    <SetupStatusMissing v-else-if="server" :loading="loading" />
    <template v-else>
      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <ReadOnlyRow :label="$t('setup.actions.actions-label')" :value="onOff(config.enableActions)" />
        <ReadOnlyRow :label="$t('setup.actions.shell-label')" :value="onOff(config.enableShell)" />
      </div>
      <p class="text-base-content/40 text-xs">{{ $t("settings.set-by-flags") }}</p>
    </template>
  </SettingsSection>
</template>

<script lang="ts" setup>
// The layout fetches the status. Swarm and Kubernetes have no setup API, so they get
// what this process started with, read-only.
const { status, loading } = useSetup();
const { t } = useI18n();
const server = config.mode === "server";
const onOff = (on: boolean) => (on ? t("setup.restart.on") : t("setup.restart.off"));

const unlockedEnvs = computed(() => {
  const locked = status.value?.locked;
  return [
    ...(locked?.authProvider ? [] : ["DOZZLE_AUTH_PROVIDER"]),
    ...(locked?.enableActions ? [] : ["DOZZLE_ENABLE_ACTIONS"]),
    ...(locked?.enableShell ? [] : ["DOZZLE_ENABLE_SHELL"]),
  ];
});
</script>
