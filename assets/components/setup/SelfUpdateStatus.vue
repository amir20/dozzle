<template>
  <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
    <div class="flex flex-wrap items-center gap-3 p-4">
      <div
        class="shrink-0 rounded-full p-2"
        :class="hasRelease ? 'bg-warning/10 text-warning' : 'bg-info/10 text-info'"
      >
        <mdi:package-down v-if="hasRelease" class="size-5" />
        <mdi:autorenew v-else class="size-5" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="text-sm font-medium">
          {{
            hasRelease ? $t("settings.new-version", { version: latestRelease?.name }) : $t("settings.auto-update-on")
          }}
        </div>
        <div class="text-base-content/60 mt-0.5 text-xs">{{ schedule }}</div>
      </div>
      <button
        v-if="canUpdate"
        type="button"
        class="btn btn-sm shrink-0"
        :disabled="busy"
        @click="updateNow(autoUpdate.image)"
        :aria-busy="busy"
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
import type { SetupAutoUpdate, SetupStatus } from "@/composable/setup/setup";

// The parent only mounts this when auto-update is on, so the panel is about what
// the schedule will do, not about turning it on. Changing the schedule is still
// the wizard's job.
const { status, autoUpdate } = defineProps<{ status: SetupStatus; autoUpdate: SetupAutoUpdate }>();

const { t } = useI18n();
const { latestRelease, hasRelease } = useAnnouncements();
// No resume marker: nothing reopens the wizard after an update started from here.
const { phase, progress, error, errorDetail, busy, updateNow } = useSelfUpdate();

const canUpdate = computed(() => canSelfUpdate(status));
const schedule = computed(() =>
  autoUpdate.mode === "daily"
    ? t("settings.auto-update-daily", { time: autoUpdate.time })
    : t("settings.auto-update-weekly", { time: autoUpdate.time }),
);
</script>
