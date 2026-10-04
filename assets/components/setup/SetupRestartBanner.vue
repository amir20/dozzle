<template>
  <!-- On every settings page, so a change saved on one page is never forgotten on
       another. The same restart as the wizard's last step. -->
  <SetupRestarting
    v-if="phase !== 'idle'"
    :timed-out="phase === 'timeout'"
    :hint="$t('setup.restart.restarting-hint')"
  />
  <InlineNotice v-else-if="count > 0" type="info">
    <span class="font-medium">{{ $t("settings.restart-pending", count) }}</span>
    <SetupPendingList :status="status" compact class="mt-0.5" />
    <div v-if="!allowed" class="mt-3 flex flex-col gap-2">
      <!-- Dozzle can't restart itself here: the same lines the wizard hands over,
           for the compose file. -->
      <template v-if="!status.canRestart">
        <p>{{ $t("setup.restart.manual") }}</p>
        <SetupSnippet :code="setupEnvSnippet(status)" />
      </template>
      <!-- It can, but not for this account: say why, as the toggles do. -->
      <p v-else>
        {{ $t(setupAccessMessageKey(status)) }}
      </p>
    </div>
    <p v-if="error" class="text-error mt-2 text-xs" role="alert">{{ error }}</p>
    <template v-if="allowed" #actions>
      <button type="button" class="btn btn-primary btn-sm" @click="restartNow({ to: route.fullPath })">
        <mdi:restart class="size-4" />
        {{ $t("setup.restart.button") }}
      </button>
    </template>
  </InlineNotice>
</template>

<script lang="ts" setup>
import type { SetupStatus } from "@/composable/setup/setup";

const { status } = defineProps<{ status: SetupStatus }>();

const route = useRoute();
const { phase, error, restartNow } = useSetupRestart();

// Every row the list below shows, the schedule included: this restart is what starts it.
const count = computed(() => setupPendingChanges(status).length);
const allowed = computed(() => setupCanRestartNow(status));
</script>
