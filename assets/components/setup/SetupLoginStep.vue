<template>
  <div>
    <h2 class="text-2xl font-bold">{{ $t("setup.login.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.login.subtitle") }}</p>

    <!-- Restarting: the page is about to go away, so nothing else competes with it. -->
    <div v-if="phase === 'restarting' || phase === 'timeout'" class="mt-6">
      <SetupRestarting :timed-out="phase === 'timeout'" :hint="$t('setup.login.restart-hint')" />
    </div>

    <!-- /data is not a volume: anything written now is gone on the next recreate. -->
    <div v-else-if="!status.dataPersisted" class="mt-6 flex flex-col gap-4">
      <div class="flex items-start gap-3">
        <div class="bg-warning/10 text-warning shrink-0 rounded-full p-2">
          <mdi:harddisk-remove class="size-5" />
        </div>
        <div class="min-w-0">
          <div class="text-sm font-semibold">{{ $t("setup.login.no-data-title") }}</div>
          <p class="text-base-content/60 mt-0.5 text-sm">{{ $t("setup.login.no-data-body") }}</p>
        </div>
      </div>
      <SetupSnippet :code="volumeSnippet" />
      <div>
        <button type="button" class="btn btn-sm" :disabled="loading" @click="fetchStatus">
          <span v-if="loading" class="loading loading-spinner loading-xs"></span>
          <mdi:refresh v-else class="size-4" />
          {{ $t("setup.login.check-again") }}
        </button>
      </div>
    </div>

    <!-- The same choice as Settings → Security. Next saves it and restarts. -->
    <SetupLoginForm v-else ref="form" :status="status" class="mt-6" @submit="submit" />

    <InlineNotice v-if="error" type="error" class="mt-4">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { SetupNextResult, SetupStatus, SetupStepId } from "@/composable/setup/setup";
import SetupLoginForm from "./SetupLoginForm.vue";

const { status, nextStep } = defineProps<{ status: SetupStatus; nextStep?: SetupStepId }>();

const { t } = useI18n();
const { loading, fetchStatus, restart, waitForRestart } = useSetup();
const form = useTemplateRef<InstanceType<typeof SetupLoginForm>>("form");

const phase = ref<"idle" | "restarting" | "timeout">("idle");
const error = ref("");

const volumeSnippet = `services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
volumes:
  dozzle_data:`;

// Login is turned on before anything else, so a save restarts straight away and
// the wizard picks up at the next step once the user has signed in.
async function restartNow(): Promise<SetupNextResult> {
  if (!status.canRestart) return "advance";
  writeSetupResume(nextStep ?? "restart");
  phase.value = "restarting";
  try {
    await restart();
  } catch (e) {
    clearSetupResume();
    phase.value = "idle";
    error.value = t(setupLoginErrorKey(e));
    return "stay";
  }
  if (!(await waitForRestart())) phase.value = "timeout";
  return "stay";
}

async function submit() {
  if (nextDisabled.value) return;
  await next();
}

async function next(): Promise<SetupNextResult> {
  error.value = "";
  if (!status.dataPersisted) return "stay";
  if (status.authProvider !== "none") return "advance";
  if (status.pending.authProvider) {
    return status.canRestart ? restartNow() : "advance";
  }
  if (!form.value?.savable) return "skip";
  if (!(await form.value.save())) return "stay";
  // Without a restart endpoint the step shows the manual instructions and waits.
  return status.canRestart ? restartNow() : "stay";
}

const method = computed(() => form.value?.method);
const saving = computed(() => !!form.value?.saving);

// While choosing, Next is the form's own save button.
const nextLabel = computed(() => {
  if (!status.dataPersisted || status.authProvider !== "none") return t("setup.next");
  if (status.pending.authProvider) return status.canRestart ? t("setup.restart.button") : t("setup.next");
  return form.value?.saveLabel || t("setup.next");
});

const nextDisabled = computed(() => {
  if (phase.value !== "idle" || saving.value) return true;
  if (!status.dataPersisted) return true;
  if (status.authProvider !== "none" || status.pending.authProvider) return false;
  return !!form.value?.savable && !form.value.canSave;
});

const busy = computed(() => saving.value || phase.value === "restarting");

// Only while nothing is saved yet: once a login is on or pending, there is nothing to
// decline. Not on the OIDC tab either, where Next already moves on without saving and a
// second button would do the same thing.
const skipLabel = computed(() =>
  status.authProvider === "none" &&
  !status.pending.authProvider &&
  phase.value === "idle" &&
  !saving.value &&
  method.value !== "oidc"
    ? t("setup.login.skip")
    : undefined,
);

defineExpose({ nextLabel, nextDisabled, nextPlain: false, skipLabel, busy, next });
</script>
