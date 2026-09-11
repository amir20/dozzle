<template>
  <div class="flex items-start gap-3" :class="compact ? 'p-3' : 'p-4'">
    <!-- Severity rides the icon. The row behind it stays neutral so the
         headline keeps full contrast. -->
    <div class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full" :class="tint">
      <mdi:alert-circle-outline class="size-4" />
    </div>

    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
        <span class="text-sm font-semibold wrap-anywhere">{{ alert.headline }}</span>
        <span class="status-pill" :class="pill">{{ alert.level || "info" }}</span>
        <span v-if="alert.isOrigin === false" class="text-base-content/40 text-xs">
          {{ $t("notifications.history.still-happening") }}
        </span>
      </div>

      <div class="text-base-content/60 mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
        <!-- In a panel already scoped to what is on screen, naming the container
             on every row is noise the reader has to skip. -->
        <span v-if="!hideContainer" class="font-mono">{{ containerName }}</span>
        <RelativeTime :date="firedAt" />
        <span class="font-mono">
          {{ $t("notifications.history.events", { n: alert.eventCount }) }}
          <span v-if="alert.suppressedCount" class="text-base-content/40">
            {{ $t("notifications.history.suppressed", { n: alert.suppressedCount }) }}
          </span>
        </span>
      </div>

      <p v-if="alert.summary" class="text-base-content/60 mt-1.5 text-sm wrap-anywhere">{{ alert.summary }}</p>

      <!-- Narrow enough that the actions cannot sit beside the text, so they sit
           under it rather than squeezing the summary into a column. -->
      <div v-if="compact" class="mt-2 flex items-center gap-1">
        <AlertRowActions :alert="alert" />
      </div>
    </div>

    <div v-if="!compact" class="flex shrink-0 items-center gap-1">
      <AlertRowActions :alert="alert" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { CloudAlert } from "@/composable/cloud/cloudAlerts";

const {
  alert,
  compact = false,
  hideContainer = false,
} = defineProps<{
  alert: CloudAlert;
  compact?: boolean;
  hideContainer?: boolean;
}>();

const { allContainersById } = storeToRefs(useContainerStore());

const firedAt = computed(() => new Date(alert.ts / 1e6));

// Falling back to a short id keeps the row readable for a container this
// instance no longer sees, rather than rendering a blank.
const containerName = computed(
  () => allContainersById.value[alert.containerId]?.name ?? alert.containerId.slice(0, 12),
);

const tint = computed(() => {
  switch (alert.level) {
    case "error":
    case "fatal":
      return "bg-error/10 text-error";
    case "warn":
      return "bg-warning/10 text-warning";
    default:
      return "bg-info/10 text-info";
  }
});

const pill = computed(() => {
  switch (alert.level) {
    case "error":
    case "fatal":
      return "status-pill-error";
    case "warn":
      return "status-pill-warning";
    default:
      return "status-pill-neutral";
  }
});
</script>
