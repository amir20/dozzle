<template>
  <!-- Update now, its progress and its outcome: the same panel on the wizard's update
       step and on Settings → Updates. Only the first row differs. -->
  <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
    <!-- The wizard: which version runs now, before any schedule exists. -->
    <div v-if="variant === 'version'" class="flex flex-wrap items-center justify-between gap-3 p-4">
      <span class="min-w-0">
        <span class="block text-sm font-medium">{{ $t("setup.update.version-label") }}</span>
        <span class="mt-0.5 block truncate font-mono text-xs">
          <span class="font-semibold">{{ autoUpdate.currentVersion }}</span>
          <span class="text-base-content/40"> · {{ autoUpdate.image }}</span>
        </span>
      </span>
      <button type="button" class="btn btn-sm shrink-0" :disabled="!canUpdate || busy" :aria-busy="busy" @click="run">
        <span v-if="phase === 'pulling'" class="loading loading-spinner loading-xs"></span>
        <mdi:download v-else class="size-4" />
        {{ $t("setup.update.update-now") }}
      </button>
    </div>

    <!-- Settings: shown only while the schedule is on, so it says what the schedule
         will do, not how to turn it on. -->
    <div v-else class="flex flex-wrap items-center gap-3 p-4">
      <div class="shrink-0 rounded-full p-2" :class="stale ? 'bg-warning/10 text-warning' : 'bg-info/10 text-info'">
        <mdi:package-down v-if="stale" class="size-5" />
        <mdi:autorenew v-else class="size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="text-sm font-medium">{{ title }}</div>
        <!-- The tag first: it is what the schedule follows and what Update now pulls. -->
        <div class="text-base-content/60 mt-0.5 truncate text-xs">
          <!-- The schedule is the row below, so only what that row cannot say. -->
          <span class="font-mono">{{ autoUpdate.image || autoUpdate.currentVersion }}</span>
        </div>
      </div>
      <button
        v-if="canUpdate"
        type="button"
        class="btn btn-sm shrink-0"
        :disabled="busy"
        :aria-busy="busy"
        @click="run"
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

    <div v-else-if="phase === 'restarting' || phase === 'timeout'" class="p-4">
      <SetupRestarting :timed-out="phase === 'timeout'" :hint="$t('setup.update.restarting-hint')" />
    </div>

    <div v-if="error" class="p-4">
      <InlineNotice type="error">
        {{ error }}
        <span v-if="errorDetail" class="text-base-content/40 mt-1 block font-mono text-xs break-all">
          {{ errorDetail }}
        </span>
      </InlineNotice>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { SetupAutoUpdate, SetupStatus, SetupStepId } from "@/composable/setup/setup";

const {
  status,
  autoUpdate,
  variant = "schedule",
  resume,
} = defineProps<{
  status: SetupStatus;
  autoUpdate: SetupAutoUpdate;
  // version: the wizard's step, before a schedule is chosen. schedule: Settings, with one on.
  variant?: "version" | "schedule";
  // The wizard step to land on again once the new version answers.
  resume?: SetupStepId;
}>();

const { t } = useI18n();
const { latestRelease } = useAnnouncements();
const { phase, progress, error, errorDetail, busy, updateNow } = useSelfUpdate({ resume });
// Only the schedule row names a newer version; the wizard has no reason to ask.
const headline = variant === "schedule" ? useSelfUpdateCheck().headline : computed(() => "current" as const);

const run = () => updateNow(autoUpdate.image);
const canUpdate = computed(() => canSelfUpdate(status));

const stale = computed(() => headline.value !== "current");
const title = computed(() => {
  if (headline.value === "image") return t("settings.update-available");
  if (headline.value === "release") return t("settings.new-version", { version: latestRelease.value?.name });
  return t("settings.auto-update-on");
});

defineExpose({ phase });
</script>
