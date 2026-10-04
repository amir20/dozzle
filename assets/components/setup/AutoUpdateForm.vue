<template>
  <!-- When and which containers: the same two questions in the setup wizard and on
       Settings → Updates. The wizard saves on Next; Settings saves as you change it. -->
  <div class="flex flex-col gap-4">
    <InlineNotice v-if="blockedReason" type="info">
      {{ blockedReason }}
      <span v-if="scheduleEditable" class="text-base-content/60 mt-1 block text-xs">
        {{ $t("auto-update.containers-still") }}
      </span>
    </InlineNotice>
    <SetupAccessNotice v-else :status="status" />

    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <div class="flex flex-wrap items-start justify-between gap-3 p-4">
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ $t("auto-update.schedule-label") }}</span>
          <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("auto-update.schedule-desc") }}</span>
          <SetupLocked v-if="status.locked.autoUpdate" env="DOZZLE_AUTO_UPDATE" class="mt-1" />
        </span>
        <span class="flex items-center gap-2">
          <select
            v-model="schedule"
            class="select select-sm w-auto"
            :aria-label="$t('auto-update.schedule-label')"
            :disabled="!canEditSchedule"
          >
            <option value="off">{{ $t("auto-update.off") }}</option>
            <option value="daily">{{ $t("auto-update.daily") }}</option>
            <option value="weekly">{{ $t("auto-update.weekly") }}</option>
          </select>
          <select
            v-model="time"
            class="select select-sm w-auto font-mono"
            :aria-label="$t('auto-update.time-label')"
            :disabled="!canEditSchedule || schedule === 'off'"
          >
            <option v-for="option in times" :key="option" :value="option">{{ option }}</option>
          </select>
        </span>
      </div>

      <fieldset class="p-4" :disabled="!canEdit">
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
            <span class="min-w-0">
              <span class="block text-sm">{{ $t(`auto-update.mode-${option}`) }}</span>
              <span class="text-base-content/60 block text-xs">{{ $t(`auto-update.mode-${option}-desc`) }}</span>
            </span>
          </label>
        </div>
      </fieldset>

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
          <button type="button" class="btn btn-sm mt-3" :disabled="!canKeepManual || keeping" @click="keepManual">
            <span v-if="keeping" class="loading loading-spinner loading-xs"></span>
            {{ $t("auto-update.keep-manual") }}
          </button>
          <p v-if="policies && !policies.persisted" class="text-base-content/40 mt-2 text-xs">
            {{ $t("auto-update.needs-volume") }}
          </p>
        </div>
      </div>
    </div>

    <InlineNotice v-if="error" type="error">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { AutoUpdateMode, SetupAutoUpdate, SetupStatus } from "@/composable/setup/setup";
import {
  UPDATE_CONTAINERS_MODES,
  type ContainerUpdatePolicy,
  type UpdateContainersMode,
  riskyContainers,
} from "@/composable/containers/updatePolicy";

const { status, autosave = false } = defineProps<{
  status: SetupStatus;
  // Save every change right away, as the rest of Settings does. The wizard saves on Next.
  autosave?: boolean;
}>();

const { t } = useI18n();
const { saveConfig } = useSetup();
const { policies, fetchPolicies, setPolicy } = useUpdatePolicies();
fetchPolicies();

const uid = useId();

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
const savedContainers = computed<UpdateContainersMode>(() => autoUpdate.value.containers ?? "picked");

const schedule = ref<AutoUpdateMode>(autoUpdate.value.mode);
const time = ref(autoUpdate.value.time || "03:00");
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

// Dozzle not being able to replace itself does not make the schedule pointless: it
// still updates the other containers. Only outside server mode does nothing run it.
const scheduleEditable = computed(() => autoUpdate.value.reason !== "not-server");

// dozzle.yml outside a volume is lost on the next recreate, so nothing is saved there.
const canEdit = computed(() => setupCanEdit(status) && scheduleEditable.value);
const canEditSchedule = computed(() => canEdit.value && !status.locked.autoUpdate);

const scheduleChanged = () =>
  schedule.value !== autoUpdate.value.mode || (schedule.value !== "off" && time.value !== autoUpdate.value.time);
const containersChanged = () => containers.value !== savedContainers.value;

const dirty = computed(() => canEdit.value && ((canEditSchedule.value && scheduleChanged()) || containersChanged()));

async function save(): Promise<boolean> {
  error.value = "";
  if (!dirty.value) return true;
  saving.value = true;
  try {
    const patch: Parameters<typeof saveConfig>[0] = {};
    if (canEditSchedule.value && scheduleChanged()) patch.autoUpdate = { mode: schedule.value, time: time.value };
    if (containersChanged()) patch.updateContainers = containers.value;
    // Applies at the next run: the scheduler reads dozzle.yml every minute.
    await saveConfig(patch);
    if (patch.updateContainers) await fetchPolicies(true);
    return true;
  } catch (e) {
    error.value = e instanceof SetupError && e.status === 409 ? t("setup.error.conflict") : t("setup.error.generic");
    return false;
  } finally {
    saving.value = false;
  }
}

if (autosave) watch([schedule, time, containers], () => save());

// What the server says now, unless something here is still waiting to be saved.
watch(autoUpdate, (saved) => {
  if (saving.value || dirty.value) return;
  schedule.value = saved.mode;
  time.value = saved.time || "03:00";
  containers.value = saved.containers ?? "picked";
});

const risky = computed(() => riskyContainers((policies.value?.containers ?? []) as ContainerUpdatePolicy[]));
const canKeepManual = computed(() => !!policies.value?.persisted && !!policies.value?.canChoose);
const keeping = ref(false);

async function keepManual() {
  keeping.value = true;
  error.value = "";
  try {
    await setPolicy(risky.value, "manual");
  } catch {
    error.value = t("auto-update.save-failed");
  } finally {
    keeping.value = false;
  }
}

defineExpose({ dirty, save, saving });
</script>
