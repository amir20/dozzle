<template>
  <div>
    <h3 class="text-base-content/60 mb-4 font-semibold tracking-wide uppercase">
      {{ $t("notifications.history.title") }}
    </h3>

    <!--
      The panel keeps its shape across states: loading, empty, failed and full
      all render the same bordered surface, so nothing reshuffles underneath the
      reader when the fetch lands.
    -->
    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <p v-if="loading && !loaded" class="text-base-content/60 p-4 text-sm">
        {{ $t("notifications.history.loading") }}
      </p>
      <p v-else-if="failed" class="text-base-content/60 p-4 text-sm">
        {{ $t("notifications.history.failed") }}
      </p>
      <!--
        An unlinked instance is not a locked feature. It gets one muted line
        naming what would be here, because alerts already show in the live
        stream and simply are not remembered without a database behind them.
      -->
      <p v-else-if="!linked" class="text-base-content/60 p-4 text-sm">
        {{ $t("notifications.history.empty-unlinked") }}
      </p>
      <p v-else-if="alerts.length === 0" class="text-base-content/60 p-4 text-sm">
        {{ $t("notifications.history.empty") }}
      </p>

      <div v-for="alert in alerts" :key="alert.alertId" class="flex items-start gap-3 p-4">
        <!-- Severity rides the icon. The row behind it stays neutral so the
             headline keeps full contrast. -->
        <div class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full" :class="tint(alert.level)">
          <mdi:alert-circle-outline class="size-4" />
        </div>

        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
            <span class="text-sm font-semibold">{{ alert.headline }}</span>
            <span class="status-pill" :class="pill(alert.level)">{{ alert.level || "info" }}</span>
            <span v-if="alert.isOrigin === false" class="text-base-content/40 text-xs">
              {{ $t("notifications.history.still-happening") }}
            </span>
          </div>

          <div class="text-base-content/60 mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
            <span class="font-mono">{{ containerName(alert.containerId) }}</span>
            <RelativeTime :date="new Date(alert.ts / 1e6)" />
            <span class="font-mono">
              {{ $t("notifications.history.events", { n: alert.eventCount }) }}
              <span v-if="alert.suppressedCount" class="text-base-content/40">
                {{ $t("notifications.history.suppressed", { n: alert.suppressedCount }) }}
              </span>
            </span>
          </div>

          <p v-if="alert.summary" class="text-base-content/60 mt-1.5 text-sm">{{ alert.summary }}</p>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <!-- The button Cloud cannot have. Cloud links out to its own page;
               here the same alert opens the real stream at the moment it fired. -->
          <button
            type="button"
            class="btn btn-ghost btn-xs"
            :title="$t('notifications.history.show-lines')"
            @click="jumpTo({ containerId: alert.containerId, date: new Date(alert.ts / 1e6), logId: alert.logId })"
          >
            <material-symbols:eye-tracking class="size-4" />
            <span class="hidden sm:inline">{{ $t("notifications.history.show-lines") }}</span>
          </button>
          <a
            v-if="alert.url"
            :href="alert.url"
            target="_blank"
            rel="noreferrer noopener"
            class="btn btn-ghost btn-xs btn-square"
            :title="$t('notifications.history.open-in-cloud')"
          >
            <mdi:open-in-new class="size-3.5 opacity-40" />
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { alerts, fetchRecentAlerts, loading, loaded, failed } = useRecentAlerts();
const { linked } = useCloudSurface();
const { jumpTo } = useLogJump();
const { allContainersById } = storeToRefs(useContainerStore());

onMounted(() => fetchRecentAlerts());

function containerName(id: string) {
  // Falling back to a short id keeps the row readable for a container this
  // instance no longer sees, rather than rendering a blank.
  return allContainersById.value[id]?.name ?? id.slice(0, 12);
}

const tint = (level: string) => {
  switch (level) {
    case "error":
    case "fatal":
      return "bg-error/10 text-error";
    case "warn":
      return "bg-warning/10 text-warning";
    default:
      return "bg-info/10 text-info";
  }
};

const pill = (level: string) => {
  switch (level) {
    case "error":
    case "fatal":
      return "status-pill-error";
    case "warn":
      return "status-pill-warning";
    default:
      return "status-pill-neutral";
  }
};
</script>
