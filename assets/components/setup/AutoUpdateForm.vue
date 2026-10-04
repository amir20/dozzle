<template>
  <!-- The auto-update schedule: the same form in the setup wizard and on Settings → Updates.
       The wizard saves on Next; Settings saves as you change it. -->
  <div class="flex flex-col gap-4">
    <InlineNotice v-if="blockedReason" type="info">
      {{ blockedReason }}
      <i18n-t
        v-if="scheduleEditable"
        keypath="setup.update.containers-still"
        tag="span"
        class="text-base-content/60 mt-1 block text-xs"
      >
        <template #label>
          <code class="font-mono">dev.dozzle.auto-update=true</code>
        </template>
      </i18n-t>
    </InlineNotice>
    <SetupAccessNotice v-else :status="status" />

    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <label class="flex items-start justify-between gap-3 p-4">
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ $t("setup.update.auto-label") }}</span>
          <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.update.auto-desc") }}</span>
          <i18n-t keypath="setup.update.auto-containers" tag="span" class="text-base-content/40 mt-1 block text-xs">
            <template #label>
              <code class="font-mono">dev.dozzle.auto-update=true</code>
            </template>
          </i18n-t>
          <SetupLocked v-if="status.locked.autoUpdate" env="DOZZLE_AUTO_UPDATE" class="mt-1" />
        </span>
        <input
          v-model="enabled"
          type="checkbox"
          class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
          :disabled="!canEdit || saving"
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
            :disabled="!canEdit || !enabled || saving"
          >
            <option value="daily">{{ $t("setup.update.daily") }}</option>
            <option value="weekly">{{ $t("setup.update.weekly") }}</option>
          </select>
          <select
            v-model="time"
            class="select select-sm w-auto font-mono"
            :aria-label="$t('setup.update.time-label')"
            :disabled="!canEdit || !enabled || saving"
          >
            <option v-for="option in times" :key="option" :value="option">{{ option }}</option>
          </select>
        </span>
      </div>
    </div>

    <InlineNotice v-if="error" type="error">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { AutoUpdateMode, SetupAutoUpdate, SetupStatus } from "@/composable/setup/setup";

const { status, autosave = false } = defineProps<{
  status: SetupStatus;
  // Save every change right away, as the rest of Settings does. The wizard saves on Next.
  autosave?: boolean;
}>();

const { t } = useI18n();
const { saveConfig } = useSetup();

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
const error = ref("");

// actions-off is not a blocker: the schedule can be saved now and starts working once
// actions are on.
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

// Dozzle not being able to replace itself does not make the schedule pointless: the
// scheduler runs in every server-mode process and updates labelled containers first,
// so a pinned tag or a binary outside docker still gets them updated. Only outside
// server mode does nothing run it.
const scheduleEditable = computed(() => autoUpdate.value.reason !== "not-server");

// dozzle.yml outside a volume is lost on the next recreate, so nothing is saved there.
const canEdit = computed(() => setupCanEdit(status) && !status.locked.autoUpdate && scheduleEditable.value);

const mode = computed<AutoUpdateMode>(() => (enabled.value ? schedule.value : "off"));

const changed = () => mode.value !== autoUpdate.value.mode || (enabled.value && time.value !== autoUpdate.value.time);

const dirty = computed(() => canEdit.value && changed());

async function save(): Promise<boolean> {
  // Putting a failed value back runs this again with nothing to save, and that must
  // not clear the error it is showing.
  if (!dirty.value) return true;
  error.value = "";
  saving.value = true;
  try {
    // Applies right away: the scheduler re-reads dozzle.yml every minute.
    await saveConfig({ autoUpdate: { mode: mode.value, time: time.value } });
    return true;
  } catch (e) {
    error.value = e instanceof SetupError && e.status === 409 ? t("setup.error.conflict") : t("setup.error.generic");
    // A control never shows a value that is not saved.
    if (autosave) reset(autoUpdate.value);
    return false;
  } finally {
    saving.value = false;
  }
}

if (autosave) watch([enabled, schedule, time], () => save());

// What the server says now, unless something here is still waiting to be saved.
function reset(saved: SetupAutoUpdate) {
  enabled.value = saved.mode !== "off";
  if (saved.mode !== "off") schedule.value = saved.mode;
  time.value = saved.time || "03:00";
}

watch(autoUpdate, (saved) => {
  if (saving.value || dirty.value) return;
  reset(saved);
});

defineExpose({ dirty, save, saving });
</script>
