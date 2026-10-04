<template>
  <div>
    <template v-if="!setupHasPending(status) && anythingSet">
      <div class="bg-success/10 text-success w-fit rounded-full p-2.5">
        <mdi:check class="size-6" />
      </div>
      <h2 class="mt-4 text-2xl font-bold">{{ $t("setup.restart.done-title") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.restart.done-body") }}</p>
    </template>

    <!-- Every step passed by: no check mark, nothing is set up to celebrate. -->
    <template v-else-if="!setupHasPending(status)">
      <div class="bg-base-content/10 text-base-content/60 w-fit rounded-full p-2.5">
        <mdi:skip-next-outline class="size-6" />
      </div>
      <h2 class="mt-4 text-2xl font-bold">{{ $t("setup.restart.nothing-title") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.restart.nothing-body") }}</p>
    </template>

    <template v-else>
      <h2 class="text-2xl font-bold">{{ $t("setup.restart.title") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.restart.subtitle") }}</p>

      <div v-if="phase !== 'idle'" class="mt-6">
        <SetupRestarting :timed-out="phase === 'timeout'" :hint="$t('setup.restart.restarting-hint')" />
      </div>

      <template v-else>
        <SetupPendingList :status="status" class="mt-6" />

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

// Whether anything is set up at all, so passing every step by does not end on "all set".
const { status, anythingSet = true } = defineProps<{ status: SetupStatus; anythingSet?: boolean }>();
const emit = defineEmits<{ seen: [] }>();

const { t } = useI18n();
// The same restart as the banner on every settings page.
const { phase, error, restartNow } = useSetupRestart();

const allowed = computed(() => setupCanRestartNow(status));

async function next(): Promise<SetupNextResult> {
  if (!setupHasPending(status) || !allowed.value) return "finish";
  // Nothing left to resume: the wizard is done once this lands.
  clearSetupResume();
  emit("seen");
  await restartNow();
  return "stay";
}

const nextLabel = computed(() =>
  setupHasPending(status) && allowed.value ? t("setup.restart.button") : t("setup.restart.done"),
);
const nextDisabled = computed(() => phase.value !== "idle");
const busy = computed(() => phase.value === "restarting");

defineExpose({ nextLabel, nextDisabled, nextPlain: false, busy, next });
</script>
