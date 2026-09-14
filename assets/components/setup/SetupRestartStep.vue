<template>
  <div>
    <template v-if="!setupHasPending(status)">
      <div class="bg-success/10 text-success w-fit rounded-full p-2.5">
        <mdi:check class="size-6" />
      </div>
      <h2 class="mt-4 text-2xl font-bold">{{ $t("setup.restart.done-title") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.restart.done-body") }}</p>
    </template>

    <template v-else>
      <h2 class="text-2xl font-bold">{{ $t("setup.restart.title") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.restart.subtitle") }}</p>

      <div v-if="phase !== 'idle'" class="mt-6">
        <SetupRestarting :timed-out="phase === 'timeout'" :hint="$t('setup.restart.restarting-hint')" />
      </div>

      <template v-else>
        <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 mt-6 divide-y rounded-lg border">
          <div v-for="change in changes" :key="change.key" class="flex items-center justify-between gap-3 p-4">
            <span class="text-sm">{{ change.label }}</span>
            <span class="font-mono text-xs font-semibold">{{ change.value }}</span>
          </div>
        </div>

        <div v-if="!allowed" class="mt-4 flex flex-col gap-3">
          <p class="text-sm">{{ $t("setup.restart.manual") }}</p>
          <SetupSnippet :code="setupEnvSnippet(status)" />
        </div>

        <InlineNotice v-if="error" type="error" class="mt-4">{{ error }}</InlineNotice>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { SetupNextResult, SetupStatus } from "@/composable/setup/setup";

const { status } = defineProps<{ status: SetupStatus }>();
const emit = defineEmits<{ seen: [] }>();

const { t } = useI18n();
const { restart, waitForRestart } = useSetup();

const phase = ref<"idle" | "restarting" | "timeout">("idle");
const error = ref("");

// Mirrors the server's rule for POST /api/setup/restart.
const allowed = computed(
  () => status.canRestart && (status.canWrite || (status.authProvider === "none" && !!status.pending.authProvider)),
);

const changes = computed(() => {
  const { authProvider, enableActions, enableShell } = status.pending;
  const on = (v: boolean) => (v ? t("setup.restart.on") : t("setup.restart.off"));
  const rows: { key: string; label: string; value: string }[] = [];
  if (authProvider != null) rows.push({ key: "auth", label: t("setup.restart.change-auth"), value: authProvider });
  if (enableActions != null)
    rows.push({ key: "actions", label: t("setup.actions.actions-label"), value: on(enableActions) });
  if (enableShell != null) rows.push({ key: "shell", label: t("setup.actions.shell-label"), value: on(enableShell) });
  return rows;
});

async function next(): Promise<SetupNextResult> {
  if (!setupHasPending(status) || !allowed.value) return "finish";
  error.value = "";
  // Nothing left to resume: the wizard is done once this lands.
  clearSetupResume();
  emit("seen");
  phase.value = "restarting";
  try {
    await restart();
  } catch {
    phase.value = "idle";
    error.value = t("setup.error.generic");
    return "stay";
  }
  if (!(await waitForRestart())) phase.value = "timeout";
  return "stay";
}

const nextLabel = computed(() =>
  setupHasPending(status) && allowed.value ? t("setup.restart.button") : t("setup.restart.done"),
);
const nextDisabled = computed(() => phase.value !== "idle");
const busy = computed(() => phase.value === "restarting");

defineExpose({ nextLabel, nextDisabled, nextPlain: false, busy, next });
</script>
