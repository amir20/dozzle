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
      <InlineNotice v-if="blockedReason" type="info" class="mt-6">{{ blockedReason }}</InlineNotice>
      <InlineNotice v-else-if="!status.dataPersisted" type="warning" class="mt-6">
        {{ $t("setup.error.no-data") }}
      </InlineNotice>
      <InlineNotice v-else-if="!status.canWrite" type="info" class="mt-6">
        {{ status.authProvider === "none" ? $t("setup.actions.window-closed") : $t("setup.actions.no-access") }}
      </InlineNotice>

      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 mt-6 divide-y rounded-lg border">
        <label class="flex items-start justify-between gap-3 p-4">
          <span class="min-w-0 flex-1">
            <span class="block text-sm font-medium">{{ $t("setup.update.auto-label") }}</span>
            <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.update.auto-desc") }}</span>
            <span v-if="status.locked.autoUpdate" class="text-base-content/40 mt-1 flex items-center gap-1 text-xs">
              <mdi:lock-outline class="size-3.5" />
              {{ $t("setup.actions.locked", { env: "DOZZLE_AUTO_UPDATE" }) }}
            </span>
          </span>
          <input
            v-model="enabled"
            type="checkbox"
            class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
            :disabled="!canEdit"
          />
        </label>

        <div class="flex flex-wrap items-center justify-between gap-3 p-4">
          <span class="min-w-0">
            <span class="block text-sm font-medium" :class="{ 'text-base-content/40': !enabled }">
              {{ $t("setup.update.when-label") }}
            </span>
            <span class="text-base-content/40 mt-0.5 block text-xs">{{ $t("setup.update.server-time") }}</span>
          </span>
          <span class="flex items-center gap-2">
            <select
              v-model="schedule"
              class="select select-sm w-auto"
              :aria-label="$t('setup.update.when-label')"
              :disabled="!canEdit || !enabled"
            >
              <option value="daily">{{ $t("setup.update.daily") }}</option>
              <option value="weekly">{{ $t("setup.update.weekly") }}</option>
            </select>
            <select
              v-model="time"
              class="select select-sm w-auto font-mono"
              :aria-label="$t('setup.update.time-label')"
              :disabled="!canEdit || !enabled"
            >
              <option v-for="option in times" :key="option" :value="option">{{ option }}</option>
            </select>
          </span>
        </div>

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
import type { AutoUpdateMode, SetupAutoUpdate, SetupNextResult, SetupStatus } from "@/composable/setup/setup";

const { status } = defineProps<{ status: SetupStatus }>();

const { t } = useI18n();
const { saveConfig } = useSetup();
// Coming back from the restart lands on this step again, showing the version it now runs.
const { phase, progress, error, errorDetail, updateNow: runUpdate } = useSelfUpdate({ resume: "update" });

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

const enabled = ref(autoUpdate.value.mode !== "off");
// Turning the toggle on picks weekly unless the file already had a schedule.
const schedule = ref<Exclude<AutoUpdateMode, "off">>(autoUpdate.value.mode === "daily" ? "daily" : "weekly");
const time = ref(autoUpdate.value.time || "03:00");
const times = computed(() => setupUpdateTimes(autoUpdate.value.time));

const saving = ref(false);

// actions-off is not a blocker here: the step only shows when actions are on or about
// to be, so the schedule can be saved now and starts working after the restart.
const blocked = computed(() => !autoUpdate.value.supported && autoUpdate.value.reason !== "actions-off");

const blockedReason = computed(() => {
  if (!blocked.value) return "";
  switch (autoUpdate.value.reason) {
    case "pinned-tag":
      return t("setup.update.reason-pinned-tag");
    case "swarm-worker":
      return t("setup.update.reason-swarm-worker");
    default:
      return t("setup.update.reason-unsupported");
  }
});

// dozzle.yml outside a volume is lost on the next recreate, so nothing is saved there.
const canEdit = computed(() => status.dataPersisted && status.canWrite && !status.locked.autoUpdate && !blocked.value);

const canUpdateNow = computed(() => canSelfUpdate(status) && phase.value !== "restarting");

const mode = computed<AutoUpdateMode>(() => (enabled.value ? schedule.value : "off"));

const updateNow = () => runUpdate(autoUpdate.value.image);

function changed() {
  return mode.value !== autoUpdate.value.mode || (enabled.value && time.value !== autoUpdate.value.time);
}

async function next(): Promise<SetupNextResult> {
  error.value = "";
  errorDetail.value = "";
  if (!canEdit.value || !changed()) return "advance";

  saving.value = true;
  try {
    // Applies right away: the scheduler re-reads dozzle.yml every minute.
    await saveConfig({ autoUpdate: { mode: mode.value, time: time.value } });
    return "advance";
  } catch (e) {
    error.value = e instanceof SetupError && e.status === 409 ? t("setup.error.conflict") : t("setup.error.generic");
    return "stay";
  } finally {
    saving.value = false;
  }
}

const nextLabel = computed(() => t("setup.next"));
const nextDisabled = computed(() => saving.value || phase.value === "pulling" || phase.value === "restarting");
const busy = computed(() => saving.value || phase.value === "pulling" || phase.value === "restarting");
// No separate skip: the toggle is the choice, and leaving it off then Next already
// declines. A "Not now" beside Next would do exactly the same thing.
const dirty = computed(() => canEdit.value && changed());

defineExpose({ nextLabel, nextDisabled, nextPlain: false, dirty, busy, next });
</script>
