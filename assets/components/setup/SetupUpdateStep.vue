<template>
  <div>
    <span class="status-pill status-pill-success">
      {{ status.enableActions ? $t("setup.update.pill-on") : $t("setup.update.pill-pending") }}
    </span>
    <h2 class="mt-3 text-2xl font-bold">{{ $t("setup.update.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.update.subtitle") }}</p>

    <div v-if="phase === 'restarting' || phase === 'timeout'" class="mt-6">
      <SetupRestarting :timed-out="phase === 'timeout'" :hint="$t('setup.update.restarting-hint')" />
    </div>

    <template v-else>
      <!-- The same form as Settings → Updates, saved on Next instead of on change. -->
      <AutoUpdateForm ref="form" :status="status" class="mt-6" />

      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 mt-4 divide-y rounded-lg border">
        <div class="flex flex-wrap items-center justify-between gap-3 p-4">
          <span class="min-w-0">
            <span class="block text-sm font-medium">{{ $t("setup.update.version-label") }}</span>
            <span class="mt-0.5 block truncate font-mono text-xs">
              <span class="font-semibold">{{ autoUpdate.currentVersion }}</span>
              <span class="text-base-content/40"> · {{ autoUpdate.image }}</span>
            </span>
          </span>
          <button
            type="button"
            class="btn btn-sm shrink-0"
            :disabled="!canUpdateNow || phase === 'pulling'"
            @click="updateNow"
          >
            <span v-if="phase === 'pulling'" class="loading loading-spinner loading-xs"></span>
            <mdi:download v-else class="size-4" />
            {{ $t("setup.update.update-now") }}
          </button>
        </div>

        <div v-if="phase === 'pulling'" class="p-4">
          <div class="flex items-center justify-between text-xs">
            <span class="text-base-content/60">{{ $t("setup.update.pulling") }}</span>
            <span v-if="progress !== undefined" class="font-mono">{{ Math.round(progress) }}%</span>
          </div>
          <div class="bg-base-content/10 mt-2 h-1.5 overflow-hidden rounded-full">
            <div
              class="bg-primary h-full rounded-full transition-[width] duration-500 motion-reduce:transition-none"
              :style="{ width: `${progress ?? 0}%` }"
            ></div>
          </div>
        </div>

        <div v-else-if="phase === 'up-to-date'" class="flex items-center gap-3 p-4">
          <div class="bg-success/10 text-success shrink-0 rounded-full p-1.5">
            <mdi:check class="size-4" />
          </div>
          <p class="text-sm">{{ $t("setup.update.up-to-date") }}</p>
        </div>
      </div>

      <p v-if="!status.enableActions" class="text-base-content/40 mt-3 text-xs">
        {{ $t("setup.update.needs-actions") }}
      </p>

      <InlineNotice v-if="error" type="error" class="mt-4">
        {{ error }}
        <span v-if="errorDetail" class="text-base-content/40 mt-1 block font-mono text-xs break-all">
          {{ errorDetail }}
        </span>
      </InlineNotice>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { SetupAutoUpdate, SetupNextResult, SetupStatus } from "@/composable/setup/setup";
import AutoUpdateForm from "./AutoUpdateForm.vue";

const { status } = defineProps<{ status: SetupStatus }>();

const { t } = useI18n();
// Coming back from the restart lands on this step again, showing the version it now runs.
const { phase, progress, error, errorDetail, updateNow: runUpdate } = useSelfUpdate({ resume: "update" });

const form = useTemplateRef<InstanceType<typeof AutoUpdateForm>>("form");

// The wizard only mounts this step when the server reports autoUpdate.
const autoUpdate = computed<SetupAutoUpdate>(
  () =>
    status.autoUpdate ?? {
      mode: "off",
      time: "03:00",
      supported: false,
      reason: "not-server",
      image: "",
      currentVersion: "",
    },
);

const canUpdateNow = computed(() => canSelfUpdate(status) && phase.value !== "restarting");
const updateNow = () => runUpdate(autoUpdate.value.image);

async function next(): Promise<SetupNextResult> {
  error.value = "";
  errorDetail.value = "";
  return (await form.value?.save()) === false ? "stay" : "advance";
}

const saving = computed(() => !!form.value?.saving);
const nextLabel = computed(() => t("setup.next"));
const nextDisabled = computed(() => saving.value || phase.value === "pulling" || phase.value === "restarting");
const busy = computed(() => saving.value || phase.value === "pulling" || phase.value === "restarting");
// No separate skip: the form is the choice, and leaving the schedule off then Next
// already declines. A "Not now" beside Next would do exactly the same thing.
const dirty = computed(() => !!form.value?.dirty);

defineExpose({ nextLabel, nextDisabled, nextPlain: false, dirty, busy, next });
</script>
