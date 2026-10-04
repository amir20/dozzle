<template>
  <LogItem :logEntry>
    <!-- Same shape as the alert and cloud-event rows: the chip marks the row and the
         log background is left alone. Severity rides on the verdict's icon. -->
    <div class="flex w-full min-w-0 flex-col gap-1 py-1 font-sans">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <!-- 0.62rem, as the chips on the alert and cloud-event rows beside it. -->
        <span
          class="border-info/45 text-info inline-flex shrink-0 items-center gap-1 rounded-xs border px-1.5 py-px text-[0.62rem] font-bold tracking-wider uppercase"
        >
          <component :is="kind.icon" class="size-3" />
          {{ $t(kind.label) }}
        </span>
        <span class="font-mono wrap-anywhere">
          <span class="opacity-60">{{ labels.from }}</span>
          <span class="mx-1 opacity-40">&rarr;</span>
          <span class="font-semibold">{{ labels.to }}</span>
        </span>
        <span v-if="sourceLabel" class="shrink-0 opacity-50">{{ $t(sourceLabel) }}</span>
      </div>

      <!-- Empty until Dozzle Cloud answers. -->
      <div v-if="verdict" class="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs">
        <component :is="look.icon" class="size-3.5 shrink-0" :class="look.text" />
        <span v-if="look.label" class="font-semibold">{{ $t(look.label) }}</span>
        <template v-if="verdict.reason">
          <span v-if="look.label" class="opacity-40">&middot;</span>
          <span class="wrap-anywhere opacity-70">{{ verdict.reason }}</span>
        </template>
        <span v-if="decisionLabel" class="status-pill status-pill-neutral">
          {{ $t(decisionLabel) }}
        </span>
        <!-- Rolling back is a Dozzle Cloud action: the link opens the update there. -->
        <a
          v-if="verdict.url"
          :href="verdict.url"
          target="_blank"
          rel="noopener"
          class="focus-visible:outline-primary ml-auto inline-flex shrink-0 items-center gap-1 rounded border px-2 py-0.5 text-xs leading-normal focus-visible:outline-2 focus-visible:outline-offset-1"
          :class="offerRollback ? look.tinted : 'border-base-content/20 hover:bg-base-content/10'"
        >
          <mdi:restore v-if="offerRollback" class="size-3" />
          {{ offerRollback ? $t("update-marker.roll-back") : $t("label.alert-view-in-cloud") }}
          <mdi:open-in-new class="size-3 opacity-60" />
        </a>
      </div>
    </div>
  </LogItem>
</template>

<script lang="ts" setup>
import type { Component } from "vue";
import { DeployLogEntry } from "@/models/LogEntry";
import { undecidedRegression, updateLabels, type DeployVerdictKind } from "@/models/ContainerUpdate";
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

interface VerdictLook {
  label?: string;
  icon: Component;
  text: string;
  // The Roll back link, only ever offered on a regression or a maybe.
  tinted?: string;
}

const VERDICTS: Record<DeployVerdictKind, VerdictLook> = {
  pending: { label: "update-marker.verdict.pending", icon: MdiCloudClockOutline, text: "text-info" },
  clean: { label: "update-marker.verdict.clean", icon: MdiCloudCheckOutline, text: "text-success" },
  regressed: {
    label: "update-marker.verdict.regressed",
    icon: MdiCloudAlertOutline,
    text: "text-error",
    tinted: "bg-error/20 text-error hover:bg-error/30 border-transparent font-semibold",
  },
  unsure: {
    label: "update-marker.verdict.unsure",
    icon: MdiCloudQuestionOutline,
    text: "text-warning",
    tinted: "bg-warning/20 text-warning hover:bg-warning/30 border-transparent font-semibold",
  },
  rolled_back_by_dozzle: {
    label: "update-marker.verdict.rolled_back_by_dozzle",
    icon: MdiCloudAlertOutline,
    text: "text-warning",
  },
};
// A newer Dozzle Cloud can send a kind this build does not know: no label rather
// than its key.
const UNKNOWN: VerdictLook = { icon: MdiCloudClockOutline, text: "text-info" };
const look = computed(() => (verdict.value && VERDICTS[verdict.value.verdict]) || UNKNOWN);

const SOURCES = ["schedule", "dozzle", "cloud"];
const DECISIONS = ["rolled_back", "kept"];
const decisionLabel = computed(() =>
  verdict.value?.decision && DECISIONS.includes(verdict.value.decision)
    ? `update-marker.decision.${verdict.value.decision}`
    : undefined,
);
// Only the updates Dozzle made are recorded; one only Dozzle Cloud knows of has
// no source and shows none.
const sourceLabel = computed(() =>
  SOURCES.includes(update.value.source) ? `update-marker.by.${update.value.source}` : undefined,
);

// A regression nobody has decided on yet: the link to Dozzle Cloud, where the
// rollback is, leads with it.
const offerRollback = computed(
  () => !update.value.rolledBack && update.value.source !== "rollback" && undecidedRegression(verdict.value),
);
</script>
