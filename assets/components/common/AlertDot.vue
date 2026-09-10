<template>
  <!--
    The mark that survives a reload.

    Alerts already splice into the live stream and vanish on refresh, because
    nothing local remembers a fire-and-forget notification. With Cloud's database
    behind it the same alert is still on the container tomorrow. Severity rides
    the dot and the row behind it stays neutral, so a long table stays scannable.
  -->
  <button
    v-if="alert"
    type="button"
    class="size-1.5 shrink-0 rounded-full transition-colors"
    :class="tone"
    :title="alert.headline"
    :aria-label="alert.headline"
    @click.stop.prevent="showLines"
  ></button>
</template>

<script lang="ts" setup>
const { containerId } = defineProps<{ containerId: string }>();

const { byContainer, fetchRecentAlerts } = useRecentAlerts();
const { jumpTo } = useLogJump();

// Shared across every row: one request for the whole table.
onMounted(() => fetchRecentAlerts());

const alert = computed(() => byContainer.value.get(containerId));

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

function showLines() {
  if (!alert.value) return;
  jumpTo({
    containerId: alert.value.containerId,
    date: new Date(alert.value.ts / 1e6),
    logId: alert.value.logId,
  });
}
</script>
