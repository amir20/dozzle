<template>
  <!--
    The mark that survives a reload.

    Alerts already splice into the live stream and vanish on refresh, because
    nothing local remembers a fire-and-forget notification. With Cloud's database
    behind it the same alert is still on the container tomorrow. Severity rides
    the dot and the row behind it stays neutral, so a long table stays scannable.

    A `title` attribute was all a row had to say what the dot meant: unstyleable,
    a second late, gone on touch, and with nowhere to put the one thing the
    reader actually wants next, which is the lines. This is a real panel instead.
  -->
  <Popover
    v-if="alert"
    class="flex shrink-0 items-center"
    placement="bottom-start"
    panel-class="rounded-box bg-base-200 border-base-content/10 w-72 border p-3 shadow-lg"
    @click.stop
  >
    <template #trigger>
      <button
        type="button"
        class="size-1.5 shrink-0 rounded-full transition-colors"
        :class="tone"
        :aria-label="alert.headline"
      ></button>
    </template>

    <template #default="{ close }">
      <div class="flex items-start gap-2.5">
        <div class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full" :class="tint">
          <mdi:alert-circle-outline class="size-4" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-semibold">{{ alert.headline }}</p>
          <div class="text-base-content/60 mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
            <span class="status-pill" :class="pill">{{ alert.level || "info" }}</span>
            <RelativeTime :date="firedAt" />
            <span class="font-mono">{{ $t("notifications.history.events", { n: alert.eventCount }) }}</span>
          </div>
          <p v-if="alert.summary" class="text-base-content/60 mt-2 text-sm">{{ alert.summary }}</p>
        </div>
      </div>

      <!-- The one move Dozzle can make that Cloud cannot, so it is the only
           action here and it is the primary one. -->
      <button type="button" class="btn btn-primary btn-sm mt-3 w-full" @click="showLines(close)">
        <mdi:text-search class="size-4" />
        {{ $t("notifications.history.show-lines") }}
      </button>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
const { containerId } = defineProps<{ containerId: string }>();

const { byContainer, fetchRecentAlerts } = useRecentAlerts();
const { jumpTo } = useLogJump();

// Shared across every row: one request for the whole table.
onMounted(() => fetchRecentAlerts());

const alert = computed(() => byContainer.value.get(containerId));
const firedAt = computed(() => new Date((alert.value?.ts ?? 0) / 1e6));

const tone = computed(() => {
  switch (alert.value?.level) {
    case "error":
    case "fatal":
      return "bg-error";
    case "warn":
      return "bg-warning";
    default:
      return "bg-info";
  }
});

// Same tinted circle and chip the alert rows use, so one alert looks like
// itself wherever it is shown.
const tint = computed(() => {
  switch (alert.value?.level) {
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
  switch (alert.value?.level) {
    case "error":
    case "fatal":
      return "status-pill-error";
    case "warn":
      return "status-pill-warning";
    default:
      return "status-pill-neutral";
  }
});

function showLines(close: () => void) {
  if (!alert.value) return;
  close();
  jumpTo({
    containerId: alert.value.containerId,
    date: new Date(alert.value.ts / 1e6),
    logId: alert.value.logId,
  });
}
</script>
