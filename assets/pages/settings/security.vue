<template>
  <!-- Who can sign in, and what Dozzle may do to containers: the wizard's first two
       steps, as the same components. -->
  <section class="flex flex-col gap-4">
    <div>
      <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("setup.steps.login") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("settings.login-desc") }}</p>
    </div>
    <SetupLoginForm v-if="status" :status="status" standalone />
    <SetupStatusMissing v-else-if="server" :loading="loading" />
    <div v-else class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <ReadOnlyRow :label="$t('setup.restart.change-auth')" :value="`auth: ${config.authProvider}`" />
    </div>
  </section>

  <section class="flex flex-col gap-4">
    <div>
      <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
        {{ $t("setup.steps.actions") }}
      </h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.actions.subtitle") }}</p>
    </div>
    <SetupTogglesForm v-if="status" :status="status" autosave />
    <SetupStatusMissing v-else-if="server" :loading="loading" />
    <template v-else>
      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <ReadOnlyRow :label="$t('setup.actions.actions-label')" :value="onOff(config.enableActions)" />
        <ReadOnlyRow :label="$t('setup.actions.shell-label')" :value="onOff(config.enableShell)" />
      </div>
      <p class="text-base-content/40 text-xs">{{ $t("settings.set-by-flags") }}</p>
    </template>
  </section>
</template>

<script lang="ts" setup>
// The layout fetches the status. Swarm and Kubernetes have no setup API, so they get
// what this process started with, read-only.
const { status, loading } = useSetup();
const { t } = useI18n();
const server = config.mode === "server";
const onOff = (on: boolean) => (on ? t("setup.restart.on") : t("setup.restart.off"));
</script>
