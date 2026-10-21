<template>
  <div
    v-if="schedule"
    :class="panel && 'border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border text-sm'"
  >
    <!-- The same parts in the reclaimable popover (compact) and on Settings → Updates
         (panel, laid out like the update schedule above it). The root stays one element,
         so @focusin from the popover still lands on it. -->
    <component :is="canEdit ? 'label' : 'div'" class="flex items-start justify-between gap-3" :class="{ 'p-4': panel }">
      <span class="min-w-0 flex-1">
        <span class="block font-medium">{{ $t("reclaim.schedule-label") }}</span>
        <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("reclaim.schedule-desc") }}</span>
        <SetupLocked v-if="lockedBy.length" :env="lockedBy" class="mt-1" />
      </span>
      <input
        v-if="canEdit"
        v-model="enabled"
        type="checkbox"
        class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
        :disabled="saving"
        @change="save"
      />
      <!-- Read only, so text: the toggle and the schedule fold into one value. -->
      <span v-else-if="schedule.mode === 'off'" class="text-base-content/60 shrink-0">
        {{ $t("setup.restart.off") }}
      </span>
      <span v-else class="shrink-0 text-right">
        {{ $t(`setup.update.${schedule.mode}`) }}
        <span class="text-base-content/60 block font-mono text-xs">{{ schedule.time }} {{ schedule.zone }}</span>
      </span>
    </component>

    <!-- The panel keeps its shape when off, like the update schedule; the popover
         only grows once there is something to set. -->
    <div
      v-if="canEdit && (panel || enabled)"
      class="flex flex-wrap items-center justify-between gap-3"
      :class="panel ? 'p-4' : 'mt-3'"
    >
      <span v-if="panel" class="min-w-0">
        <span class="block font-medium" :class="{ 'text-base-content/40': !enabled }">
          {{ $t("setup.update.when-label") }}
        </span>
        <span class="text-base-content/40 mt-0.5 block text-xs">{{ $t("setup.update.server-time") }}</span>
      </span>
      <span class="flex items-center gap-2">
        <select
          v-model="frequency"
          class="select select-sm w-auto"
          :aria-label="$t('setup.update.when-label')"
          :disabled="!enabled || saving"
          @change="save"
        >
          <option value="daily">{{ $t("setup.update.daily") }}</option>
          <option value="weekly">{{ $t("setup.update.weekly") }}</option>
        </select>
        <select
          v-model="time"
          class="select select-sm w-auto font-mono"
          :aria-label="$t('setup.update.time-label')"
          :disabled="!enabled || saving"
          @change="save"
        >
          <option v-for="option in times" :key="option" :value="option">{{ option }}</option>
        </select>
        <span class="text-base-content/60 font-mono text-xs">{{ schedule.zone }}</span>
      </span>
    </div>

    <div v-if="error || schedule.lastRun" :class="panel ? 'p-4' : 'mt-2'" class="space-y-1 text-xs">
      <p v-if="error" class="text-error">{{ $t("error.something-went-wrong") }}</p>
      <i18n-t v-if="schedule.lastRun" scope="global" keypath="reclaim.last-run" tag="p" class="text-base-content/40">
        <template #size>
          <span class="font-mono">{{ formatBytes(schedule.lastRun.spaceReclaimed, { decimals: 1 }) }}</span>
        </template>
        <template #when>
          <RelativeTime :date="new Date(schedule.lastRun.at)" />
        </template>
      </i18n-t>
      <p v-if="schedule.lastRun?.failed" class="text-base-content/40">
        {{ $t("reclaim.last-run-failed", { count: schedule.lastRun.failed }, schedule.lastRun.failed) }}
      </p>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { AutoUpdateMode } from "@/composable/setup/setup";
import { setupCanEditUpdates, setupUpdateTimes } from "@/composable/setup/setup";

const { panel = false } = defineProps<{
  // Settings' divided rows instead of the popover's compact stack.
  panel?: boolean;
}>();

const { status, fetchStatus, saveConfig } = useSetup();
if (!status.value) fetchStatus();

const schedule = computed(() => status.value?.autoPrune);
// The env vars that fixed the schedule, which then only shows what they say.
const lockedBy = computed(() => [
  ...(status.value?.locked.autoPruneMode ? ["DOZZLE_AUTO_CLEANUP"] : []),
  ...(status.value?.locked.autoPruneTime ? ["DOZZLE_AUTO_CLEANUP_TIME"] : []),
]);
const canEdit = computed(() => !!status.value && setupCanEditUpdates(status.value) && !lockedBy.value.length);

const enabled = ref(false);
// Turning the toggle on picks weekly unless the file already had a schedule.
const frequency = ref<Exclude<AutoUpdateMode, "off">>("weekly");
const time = ref("04:00");
// A fetch that lands after mounting, or a save from the other surface, moves the
// controls along with it.
function reset() {
  const value = schedule.value;
  if (!value) return;
  enabled.value = value.mode !== "off";
  if (value.mode !== "off") frequency.value = value.mode;
  time.value = value.time;
}
watch(schedule, reset, { immediate: true });
const times = computed(() => setupUpdateTimes(schedule.value?.time ?? time.value));

const saving = ref(false);
const error = ref(false);

async function save() {
  saving.value = true;
  error.value = false;
  try {
    await saveConfig({ autoPrune: { mode: enabled.value ? frequency.value : "off", time: time.value } });
  } catch {
    error.value = true;
    reset();
  } finally {
    saving.value = false;
  }
}
</script>
