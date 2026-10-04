<template>
  <!-- When and which containers: the same two questions in the setup wizard and on
       Settings → Updates. The wizard saves on Next; Settings saves as you change it. -->
  <div class="flex flex-col gap-4">
    <template v-if="!notices" />
    <InlineNotice v-else-if="blockedReason" type="info">
      {{ blockedReason }}
      <span v-if="scheduleEditable && containers !== 'off'" class="text-base-content/60 mt-1 block text-xs">
        {{ $t("auto-update.containers-still") }}
      </span>
    </InlineNotice>
    <SetupAccessNotice v-else :status="status" :envs="unlockedEnvs" />

    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <template v-if="canEditSchedule">
        <label class="flex items-start justify-between gap-3 p-4">
          <span class="min-w-0 flex-1">
            <span class="block text-sm font-medium">{{ $t("setup.update.auto-label") }}</span>
            <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.update.auto-desc") }}</span>
          </span>
          <input
            v-model="enabled"
            type="checkbox"
            class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
            :disabled="saving"
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
              :disabled="!enabled || saving"
            >
              <option value="daily">{{ $t("setup.update.daily") }}</option>
              <option value="weekly">{{ $t("setup.update.weekly") }}</option>
            </select>
            <select
              v-model="time"
              class="select select-sm w-auto font-mono"
              :aria-label="$t('setup.update.time-label')"
              :disabled="!enabled || saving"
            >
              <option v-for="option in times" :key="option" :value="option">{{ option }}</option>
            </select>
          </span>
        </div>
      </template>

      <!-- Read only, so text: the toggle and the schedule fold into one value. -->
      <div v-else class="flex items-start justify-between gap-3 p-4">
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ $t("setup.update.auto-label") }}</span>
          <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.update.auto-desc") }}</span>
          <SetupLocked v-if="status.locked.autoUpdate" env="DOZZLE_AUTO_UPDATE" class="mt-1" />
        </span>
        <span v-if="autoUpdate.mode === 'off'" class="text-base-content/60 shrink-0 text-sm">
          {{ $t("setup.restart.off") }}
        </span>
        <span v-else class="shrink-0 text-right text-sm">
          {{ $t(`setup.update.${autoUpdate.mode}`) }}
          <span class="text-base-content/60 block font-mono text-xs">{{ autoUpdate.time }}</span>
        </span>
      </div>

      <fieldset v-if="canEditContainers" class="p-4" :disabled="saving">
        <legend class="contents">
          <span class="block text-sm font-medium">{{ $t("auto-update.which-label") }}</span>
        </legend>
        <div class="mt-2 flex flex-col gap-0.5">
          <label
            v-for="option in UPDATE_CONTAINERS_MODES"
            :key="option"
            class="hover:bg-base-300/40 -mx-2 flex items-start gap-3 rounded-md p-2 transition-colors"
          >
            <input
              v-model="containers"
              type="radio"
              class="radio radio-primary radio-sm mt-0.5"
              :name="`update-containers-${uid}`"
              :value="option"
            />
            <span class="min-w-0 flex-1">
              <span class="block text-sm">{{ $t(`auto-update.mode-${option}`) }}</span>
              <i18n-t
                :keypath="`auto-update.mode-${option}-desc`"
                tag="span"
                class="text-base-content/60 block text-xs"
              >
                <template #label>
                  <code class="font-mono">{{ option === "all" ? OFF_LABEL : AUTO_LABEL }}</code>
                </template>
              </i18n-t>
            </span>
          </label>
        </div>
      </fieldset>

      <div v-else class="flex items-start justify-between gap-3 p-4">
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ $t("auto-update.which-label") }}</span>
          <i18n-t
            :keypath="`auto-update.mode-${savedContainers}-desc`"
            tag="span"
            class="text-base-content/60 mt-0.5 block text-xs"
          >
            <template #label>
              <code class="font-mono">{{ savedContainers === "all" ? OFF_LABEL : AUTO_LABEL }}</code>
            </template>
          </i18n-t>
          <SetupLocked v-if="status.locked.updateContainers" env="DOZZLE_UPDATE_CONTAINERS" class="mt-1" />
        </span>
        <span class="shrink-0 text-right text-sm">
          {{ $t(`auto-update.mode-${savedContainers}`) }}
        </span>
      </div>

      <!-- Everything reaches the databases too, which is the one way it goes badly. -->
      <div v-if="containers === 'all' && risky.length" class="flex items-start gap-3 p-4">
        <div class="bg-warning/10 text-warning shrink-0 rounded-full p-1.5">
          <mdi:database-alert-outline class="size-4" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm">{{ $t("auto-update.risky", risky.length) }}</p>
          <p class="text-base-content/60 mt-1 truncate font-mono text-xs">
            {{ risky.map((c) => c.name).join(", ") }}
          </p>
          <i18n-t keypath="auto-update.risky-hint" tag="p" class="text-base-content/60 mt-2 text-xs">
            <template #label>
              <code class="font-mono">{{ OFF_LABEL }}</code>
            </template>
          </i18n-t>
        </div>
      </div>
    </div>

    <InlineNotice v-if="error" type="error">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { AutoUpdateMode, SetupAutoUpdate, SetupStatus } from "@/composable/setup/setup";
import {
  DEFAULT_UPDATE_CONTAINERS_MODE,
  UPDATE_CONTAINERS_MODES,
  UPDATE_LABEL,
  type ContainerUpdatePolicy,
  type UpdateContainersMode,
  riskyContainers,
} from "@/composable/containers/updatePolicy";

const {
  status,
  autosave = false,
  notices = true,
} = defineProps<{
  status: SetupStatus;
  // False when the page shows a notice that outranks this form's own.
  notices?: boolean;
  // Save every change right away, as the rest of Settings does. The wizard saves on Next.
  autosave?: boolean;
}>();

const { t } = useI18n();
const { saveConfig } = useSetup();
const { policies, fetchPolicies } = useUpdatePolicies();
fetchPolicies();

const uid = useId();
const AUTO_LABEL = `${UPDATE_LABEL}=auto`;
const OFF_LABEL = `${UPDATE_LABEL}=off`;

const autoUpdate = computed(() => setupAutoUpdate(status));

const enabled = ref(autoUpdate.value.mode !== "off");
// Turning the toggle on picks weekly unless the file already had a schedule.
const schedule = ref<Exclude<AutoUpdateMode, "off">>(autoUpdate.value.mode === "daily" ? "daily" : "weekly");
const time = ref(autoUpdate.value.time || "03:00");
const savedContainers = computed<UpdateContainersMode>(
  () => autoUpdate.value.containers ?? DEFAULT_UPDATE_CONTAINERS_MODE,
);
const containers = ref<UpdateContainersMode>(savedContainers.value);
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
// The schedule and which containers have their own env vars, so each has its own lock.
const canEdit = computed(() => setupCanEdit(status) && scheduleEditable.value);
const canEditSchedule = computed(() => canEdit.value && !status.locked.autoUpdate);
const canEditContainers = computed(() => canEdit.value && !status.locked.updateContainers);

// What the access notice points at instead: the env vars that would still change something.
const unlockedEnvs = computed(() => [
  ...(status.locked.autoUpdate ? [] : ["DOZZLE_AUTO_UPDATE"]),
  ...(status.locked.updateContainers ? [] : ["DOZZLE_UPDATE_CONTAINERS"]),
]);

const mode = computed<AutoUpdateMode>(() => (enabled.value ? schedule.value : "off"));

const scheduleChanged = () =>
  mode.value !== autoUpdate.value.mode || (enabled.value && time.value !== autoUpdate.value.time);
const containersChanged = () => containers.value !== savedContainers.value;

const dirty = computed(
  () => (canEditSchedule.value && scheduleChanged()) || (canEditContainers.value && containersChanged()),
);

async function save(): Promise<boolean> {
  // Putting a failed value back runs this again with nothing to save, and that must
  // not clear the error it is showing.
  if (!dirty.value) return true;
  error.value = "";
  saving.value = true;
  try {
    // Applies right away: the scheduler re-reads dozzle.yml every minute.
    const patch: Parameters<typeof saveConfig>[0] = {};
    if (canEditSchedule.value && scheduleChanged()) patch.autoUpdate = { mode: mode.value, time: time.value };
    if (canEditContainers.value && containersChanged()) patch.updateContainers = containers.value;
    await saveConfig(patch);
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

if (autosave) watch([enabled, schedule, time, containers], () => save());

// What the server says now, unless something here is still waiting to be saved.
function reset(saved: SetupAutoUpdate) {
  enabled.value = saved.mode !== "off";
  if (saved.mode !== "off") schedule.value = saved.mode;
  time.value = saved.time || "03:00";
  containers.value = saved.containers ?? DEFAULT_UPDATE_CONTAINERS_MODE;
}

watch(autoUpdate, (saved) => {
  if (saving.value || dirty.value) return;
  reset(saved);
});

const risky = computed(() => riskyContainers((policies.value?.containers ?? []) as ContainerUpdatePolicy[]));

defineExpose({ dirty, save, saving });
</script>
