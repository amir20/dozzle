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
    panel-class="rounded-box bg-base-200 border-base-content/10 w-80 border shadow-lg"
    hover
    @click.stop
  >
    <template #trigger>
      <!-- The dot stays 6px, because a bigger one would read as a status column
           rather than a mark on the name. Everything that makes it usable is
           therefore off the painted box: the hit area is an invisible ::after,
           and the ring is drawn outside it. -->
      <button type="button" class="alert-dot" :class="tone" :aria-label="alert.headline"></button>
    </template>

    <template #default="{ close }">
      <!-- The same parts as an AlertRow, at popover width: tinted glyph, a
           headline that carries its own severity chip, then the meta line. -->
      <div class="flex items-start gap-3 p-3">
        <div class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full" :class="tint">
          <mdi:alert-circle-outline class="size-4" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
            <span class="text-sm font-semibold">{{ alert.headline }}</span>
            <span class="status-pill" :class="pill">{{ alert.level || "info" }}</span>
          </div>
          <div class="text-base-content/60 mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
            <RelativeTime :date="firedAt" />
            <span class="font-mono">{{ $t("notifications.history.events", { n: alert.eventCount }) }}</span>
          </div>
          <p v-if="alert.summary" class="text-base-content/60 mt-1.5 line-clamp-4 text-sm">{{ alert.summary }}</p>
        </div>
      </div>

      <!-- The one move Dozzle can make that Cloud cannot, so it is the only
           action here. A solid block at panel width shouted over the alert it
           was about, so it is a row instead: hairline above, quiet until
           hovered. -->
      <div class="bg-base-content/10 h-px"></div>
      <div class="p-1.5">
        <button
          type="button"
          class="hover:bg-base-300 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm font-medium transition-colors"
          @click="showLines(close)"
        >
          <mdi:text-search class="size-4 shrink-0 opacity-60" />
          {{ label }}
        </button>
      </div>
    </template>
  </Popover>
</template>

<script lang="ts" setup>
const { containerId } = defineProps<{ containerId: string }>();

const { byContainer, fetchRecentAlerts } = useRecentAlerts();
const { jumpTo } = useLogJump();
const { t } = useI18n();

// Shared across every row: one request for the whole table.
onMounted(() => fetchRecentAlerts());

const alert = computed(() => byContainer.value.get(containerId));
const firedAt = computed(() => new Date((alert.value?.ts ?? 0) / 1e6));

// A CPU spike or a container event has no line to show — logId is absent for
// exactly those — so promising lines sends the reader looking for something
// that was never there. The destination is the same either way; only the
// log-anchored case can point at the line itself.
const label = computed(() =>
  alert.value?.logId ? t("notifications.history.show-lines") : t("notifications.history.show-moment"),
);

// The text colour rides along so the hover ring can be drawn from currentColor
// and match whatever severity the dot is wearing.
const tone = computed(() => {
  switch (alert.value?.level) {
    case "error":
    case "fatal":
      return "bg-error text-error";
    case "warn":
      return "bg-warning text-warning";
    default:
      return "bg-info text-info";
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

<style scoped>
@reference "@/main.css";

/* A 6px target with the UA's default arrow over it is not a control anyone
   finds. Tailwind v4's preflight gives buttons `cursor: default`, so the dot
   looked painted on, and at 6px there was nothing to aim at even once you knew
   it was there. */
.alert-dot {
  @apply relative size-1.5 shrink-0 cursor-pointer rounded-full transition-[box-shadow,transform];
}

/* Roughly a 24px target without taking 24px of the row: the name beside it
   keeps its width and the dot keeps its position. */
.alert-dot::after {
  content: "";
  @apply absolute -inset-2;
}

.alert-dot:hover,
.alert-dot:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in oklab, currentColor 30%, transparent);
}

@media (prefers-reduced-motion: no-preference) {
  .alert-dot:hover {
    transform: scale(1.2);
  }
}
</style>
