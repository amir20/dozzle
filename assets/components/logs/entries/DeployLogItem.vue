<template>
  <LogItem :logEntry>
    <div class="deploy-row flex w-full min-w-0 flex-col gap-1 py-1 font-sans" :data-verdict="tint">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <span class="chip">
          <component :is="kind.icon" class="size-3" />
          {{ $t(kind.label) }}
        </span>
        <span class="font-mono wrap-anywhere">
          <span class="opacity-60">{{ labels.from }}</span>
          <span class="mx-1 opacity-40">&rarr;</span>
          <span class="font-semibold">{{ labels.to }}</span>
        </span>
        <span v-if="sourceLabel" class="shrink-0 opacity-50">{{ $t(sourceLabel) }}</span>
        <button
          v-if="canRollBack"
          type="button"
          class="act ml-auto"
          :class="{ 'act-tinted': verdict?.verdict === 'regressed' }"
          @click="requestRollback(current!)"
        >
          <mdi:restore class="size-3" />
          {{ $t("update-marker.roll-back") }}
        </button>
      </div>

      <!-- Empty until Dozzle Cloud answers. -->
      <div v-if="verdict" class="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs">
        <component :is="verdictIcon" class="verdict-icon size-3.5 shrink-0" />
        <span class="font-semibold">{{ $t(`update-marker.verdict.${verdict.verdict}`) }}</span>
        <template v-if="verdict.reason">
          <span class="opacity-40">&middot;</span>
          <span class="wrap-anywhere opacity-70">{{ verdict.reason }}</span>
        </template>
        <span v-if="verdict.decision" class="status-pill status-pill-neutral">
          {{ $t(`update-marker.decision.${verdict.decision}`) }}
        </span>
        <a v-if="verdict.url" :href="verdict.url" target="_blank" rel="noopener" class="act ml-auto">
          {{ $t("label.alert-view-in-cloud") }}
        </a>
      </div>
    </div>
  </LogItem>
</template>

<script lang="ts" setup>
import { DeployLogEntry } from "@/models/LogEntry";
import { updateLabels } from "@/models/ContainerUpdate";
import MdiUpdate from "~icons/mdi/update";
import MdiRestore from "~icons/mdi/restore";
import MdiCloudCheckOutline from "~icons/mdi/cloud-check-outline";
import MdiCloudAlertOutline from "~icons/mdi/cloud-alert-outline";
import MdiCloudQuestionOutline from "~icons/mdi/cloud-question-outline";
import MdiCloudClockOutline from "~icons/mdi/cloud-clock-outline";

const { logEntry } = defineProps<{
  logEntry: DeployLogEntry;
  showContainerName?: boolean;
}>();

const { enableActions } = config;
const store = useContainerStore();

const update = computed(() => logEntry.update);
const verdict = computed(() => logEntry.verdict);
const labels = computed(() => updateLabels(update.value));

// A rollback reads as one, and a swap Dozzle undid says so: neither is an
// update that now runs.
const kind = computed(() => {
  if (update.value.rolledBack) return { label: "update-marker.undone", icon: MdiRestore };
  if (update.value.source === "rollback") return { label: "update-marker.rolled-back", icon: MdiRestore };
  return { label: "update-marker.updated", icon: MdiUpdate };
});

const SOURCES = ["schedule", "dozzle", "cloud", "watchtower", "external"];
const sourceLabel = computed(() =>
  SOURCES.includes(update.value.source) ? `update-marker.by.${update.value.source}` : undefined,
);

// The container this update left running, while it still runs. Rolling back acts
// on it, and goes to the image its last update replaced, which is this update's
// only while this container is the one it created.
const current = computed(() => store.findContainerById(update.value.newId));
const canRollBack = computed(() => {
  const c = current.value;
  if (!enableActions || !c || c.state === "deleted" || !c.rollbackTarget) return false;
  if (update.value.rolledBack || update.value.source === "rollback") return false;
  return verdict.value?.decision !== "rolled_back";
});

const tint = computed(() => {
  switch (verdict.value?.verdict) {
    case "clean":
      return "success";
    case "regressed":
      return "error";
    case "unsure":
    case "rolled_back_by_dozzle":
      return "warn";
    default:
      return "info";
  }
});

const verdictIcon = computed(() => {
  switch (verdict.value?.verdict) {
    case "clean":
      return MdiCloudCheckOutline;
    case "regressed":
    case "rolled_back_by_dozzle":
      return MdiCloudAlertOutline;
    case "unsure":
      return MdiCloudQuestionOutline;
    default:
      return MdiCloudClockOutline;
  }
});
</script>

<style scoped>
@reference "@/main.css";

/* Same shape as the alert and cloud-event rows: the chip marks the row and the
   log background is left alone. Severity rides on the verdict's icon. Keyed on
   data-verdict, not data-level, for the reason AlertLogItem gives. */
.deploy-row {
  --tint: var(--color-info);
}
.deploy-row[data-verdict="success"] {
  --tint: var(--color-success);
}
.deploy-row[data-verdict="warn"] {
  --tint: var(--color-warning);
}
.deploy-row[data-verdict="error"] {
  --tint: var(--color-error);
}

.chip {
  background-color: transparent;
  color: var(--color-info);
  border: 1px solid color-mix(in oklab, var(--color-info) 45%, transparent);
  @apply inline-flex shrink-0 items-center gap-1 rounded-xs px-1.5 py-px text-[0.62rem] font-bold tracking-wider uppercase;
}

.verdict-icon {
  color: var(--tint);
}

.act {
  @apply inline-flex shrink-0 items-center gap-1 rounded border px-2 py-0.5 text-[0.7rem] leading-normal;
  border-color: color-mix(in oklab, var(--color-base-content) 22%, transparent);
}
.act:hover {
  background-color: color-mix(in oklab, var(--color-base-content) 10%, transparent);
}
.act:focus-visible {
  @apply outline-primary outline-2 outline-offset-1;
}
.act-tinted {
  border-color: transparent;
  background-color: color-mix(in oklab, var(--tint) 20%, transparent);
  color: var(--tint);
  @apply font-semibold;
}
.act-tinted:hover {
  background-color: color-mix(in oklab, var(--tint) 30%, transparent);
}
</style>
