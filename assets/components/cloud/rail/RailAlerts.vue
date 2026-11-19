<template>
  <!--
    What the reader's rules caught, as one timeline: a chart of how often they
    matched, then a day at a time, one row per problem. Scoped to the containers
    on screen or, widened or on a page with none, to every container.
  -->
  <div ref="root" class="h-full [scrollbar-gutter:stable] overflow-y-auto p-4">
    <div class="mb-2 flex items-center gap-2">
      <span class="text-base-content/60 truncate text-xs font-semibold tracking-wide uppercase">
        {{ $t("cloud-rail.alerts-chart") }}
      </span>
      <div role="tablist" class="tabs tabs-box tabs-xs ml-auto shrink-0">
        <button
          v-for="option in windows"
          :key="option"
          role="tab"
          class="tab"
          :class="{ 'tab-active': window === option }"
          @click="window = option"
        >
          {{ $t(`cloud-rail.window-${option}`) }}
        </button>
      </div>
    </div>

    <p v-if="loading && !loaded" class="text-base-content/60 text-sm">
      {{ $t("notifications.history.loading") }}
    </p>
    <p v-else-if="failed" class="text-base-content/60 text-sm">
      {{ $t("notifications.history.failed") }}
    </p>
    <!-- Nothing matched: say how far back that answer reaches, and where a rule
         that would match is made. -->
    <div v-else-if="!days.length" class="space-y-2 text-sm" data-empty>
      <p class="text-base-content/60">
        {{ scoped ? $t("cloud-rail.alerts-empty-here") : $t("notifications.history.empty") }}
      </p>
      <p v-if="retentionDays" class="text-base-content/40 text-xs">
        {{ $t("cloud-rail.alerts-retention", { n: retentionDays }, retentionDays) }}
      </p>
      <RouterLink
        v-if="config.enableNotifications"
        :to="{ name: '/notifications', query: { action: 'create-alert' } }"
        class="link inline-block"
        @click="closeRail()"
      >
        {{ $t("cloud-rail.create-alert") }}
      </RouterLink>
    </div>

    <template v-else>
      <!-- The hovered bar's moment, count and problem; otherwise the window's total. -->
      <p class="text-base-content/40 mb-1 truncate font-mono text-xs tabular-nums" data-readout>
        <template v-if="hover">
          {{ clock.format(new Date(hover.from / 1e6)) }} ·
          <span class="text-base-content/70 font-semibold">{{ $t("cloud-rail.alerts-matches", hover.count) }}</span>
          <template v-if="hover.problem"> · {{ hover.problem }}</template>
        </template>
        <span v-else class="text-base-content/60 font-semibold">{{ $t("cloud-rail.alerts-matches", total) }}</span>
      </p>

      <!-- Only from a Cloud that counts per bucket. Drawing it from the matches
           would stop at Cloud's cap and call a busy hour a quiet one. -->
      <div v-if="buckets" data-chart>
        <button
          type="button"
          class="block w-full"
          :aria-label="$t('cloud-rail.alerts-chart-scroll')"
          @click="scrollToHovered"
        >
          <BarChart
            :chart-data="chartData"
            bar-class="text-info"
            :hover-index="hovered?.index ?? null"
            class="h-16"
            @hover-value="(_: number, index: number, bars: number) => (hovered = { index, bars })"
            @hover-end="hovered = null"
          />
        </button>
        <!-- A dot under each bar a notification went out from, at the bar's centre
             (BarChart's columns are (width + 2px gap) / count wide). -->
        <div class="relative mt-1 h-1.5" data-notified-strip>
          <span
            v-for="dot in dots"
            :key="dot.index"
            class="absolute top-0 size-1.5 -translate-x-1/2 rounded-full"
            :class="dot.color"
            :style="{ left: `calc((100% + 2px) * ${dot.at} - 1px)` }"
            data-notified-dot
          ></span>
        </div>
        <div class="text-base-content/40 mt-1 flex justify-between font-mono text-xs tabular-nums" data-range>
          <!-- Cloud may shorten a wide window; the bars start at the first bucket. -->
          <span>{{ clock.format(new Date(buckets?.[0] ? buckets[0].startTs / 1e6 : range.from)) }}</span>
          <span>{{ clock.format(new Date(range.to)) }}</span>
        </div>
      </div>

      <div class="mt-4 space-y-6">
        <section v-for="day in days" :key="day.key">
          <h3 class="text-base-content/60 mb-2 flex items-baseline gap-2 text-xs font-semibold tracking-wide uppercase">
            {{ day.label }}
          </h3>
          <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
            <AlertTimelineRow
              v-for="row in day.rows"
              :key="row.key"
              :row="row"
              :hide-container="single"
              :highlighted="highlighted === row.key"
              :alone="day.rows.length === 1"
            />
          </div>
        </section>
      </div>
    </template>

    <!-- The chart counts every match, the rows only the ones Cloud listed. -->
    <p v-if="loaded && !failed && truncated && days.length" class="text-base-content/40 mt-3 text-xs" data-partial>
      {{ buckets ? $t("cloud-rail.alerts-partial-chart") : $t("cloud-rail.alerts-partial") }}
    </p>
    <!-- Everything above, the empty state included, is about these containers only. -->
    <p v-if="loaded && !failed && capped" class="text-base-content/40 mt-3 text-xs" data-capped>
      {{ $t("cloud-rail.alerts-capped", { n: MAX_ASKED }) }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import type { BarDataPoint } from "@/components/ui/BarChart.vue";
import type { AlertBucket, CloudAlert, CloudEvent } from "@/composable/cloud/cloudAlerts";
import {
  TIMELINE_WINDOWS,
  bucketEnd,
  buildTimeline,
  levelDot,
  MAX_BUCKETS,
  notifiedLevel,
  problemAt,
  rowAt,
  type TimelineWindow,
} from "@/composable/cloud/alertsTimeline";
import { hoverSpan, windowClock, type MetricsHover } from "@/composable/cloud/metricsTime";
import { MAX_ASKED } from "@/composable/cloud/cloudFindings";

const { markAlertsSeen, loaded: recentLoaded, fetchRecentAlerts, refreshRecentAlerts } = useRecentAlerts();
// What the rail's dot counts, in this panel's scope.
const { visible: dotted } = useViewAlerts();
onMounted(() => (recentLoaded.value ? refreshRecentAlerts() : fetchRecentAlerts()));
const { fetchAlerts } = useCloudAlerts();
const { containers: inScope } = useCloudScope();
// The containers on screen, or with none (home, or the chip widened) every
// container this instance knows. That list runs one past what Cloud takes in
// one ask, so the panel can tell it was cut; only the first MAX_ASKED are
// sent, which leaves `truncated` meaning only that Cloud cut the list.
const scope = useFindingsScope();
const capped = computed(() => scope.value.length > MAX_ASKED);
const scoped = computed(() => inScope.value.length > 0);
const single = computed(() => inScope.value.length === 1);
const { locale } = useI18n();
const { closeRail } = useCloudRail();
const { cloudStatus, ensureCloudStatus } = useCloudConfig();
const retentionDays = computed(() => cloudStatus.value?.plan.retention_days ?? 0);
onMounted(ensureCloudStatus);

// A week is only offered where the plan keeps one.
const windows = computed<TimelineWindow[]>(() =>
  retentionDays.value && retentionDays.value < 7 ? ["24h"] : ["24h", "7d"],
);
const window = ref<TimelineWindow>("24h");
watch(windows, (options) => {
  if (!options.includes(window.value)) window.value = "24h";
});

const hits = ref<CloudAlert[]>([]);
const events = ref<CloudEvent[]>([]);
const buckets = ref<AlertBucket[]>();
const truncated = ref(false);
const range = ref({ from: 0, to: 0 });
const loading = ref(false);
const loaded = ref(false);
const failed = ref(false);

const root = ref<HTMLElement | null>(null);
// One bar per 5px column, as the metrics charts draw: a fixed 24 left thin bars
// with wide gaps at drawer width. Capped where Cloud caps a call; the window's
// own count stands in until the panel has a width.
const { width: rootWidth } = useElementSize(root);
const columns = ref(0);
// Settled before refetching, so a resize does not fire a request per five pixels.
watchDebounced(rootWidth, (w) => w > 0 && (columns.value = Math.min(MAX_BUCKETS, Math.floor(w / 5))), {
  debounce: 300,
});

let request = 0;
async function load() {
  const mine = ++request;
  const ids = scope.value.slice(0, MAX_ASKED).map((c) => c.id);
  const { hours, buckets: fallback } = TIMELINE_WINDOWS[window.value];
  const count = columns.value || fallback;
  const to = Date.now();
  const from = to - hours * 3600_000;
  loading.value = true;
  failed.value = false;
  const result = await fetchAlerts(ids, new Date(from), new Date(to), {
    followUps: true,
    events: true,
    buckets: count,
    limit: 200,
  });
  if (mine !== request) return;
  hits.value = result.alerts;
  events.value = result.events;
  buckets.value = result.buckets;
  truncated.value = !!result.truncated;
  range.value = { from, to };
  failed.value = !!result.failed;
  loading.value = false;
  loaded.value = true;
}
watch([() => scope.value.map((c) => c.id).join(","), window, columns], load, { immediate: true });

// Reads the shared tick so "Today" rolls over at midnight on a page left open.
const days = computed(() => {
  void relativeTimeTick.value;
  return buildTimeline(hits.value, events.value, locale.value);
});

const clock = useMetricsClock(() => windowClock(window.value));

const chartData = computed<BarDataPoint[]>(() =>
  (buckets.value ?? []).map((b) => ({ percent: b.matched, value: b.matched })),
);
const total = computed(() =>
  buckets.value
    ? buckets.value.reduce((sum, b) => sum + b.matched, 0)
    : days.value.reduce((sum, d) => sum + d.rows.reduce((n, r) => n + r.happened, 0), 0),
);

const toNs = computed(() => range.value.to * 1e6);
const dots = computed(() =>
  (buckets.value ?? []).flatMap((b, index, all) =>
    b.notified > 0
      ? [
          {
            index,
            at: (index + 0.5) / all.length,
            color: levelDot(notifiedLevel(hits.value, b.startTs, bucketEnd(all, index, toNs.value))),
          },
        ]
      : [],
  ),
);

const hovered = ref<MetricsHover | null>(null);
/** The buckets behind the hovered bar, as one span of time. */
const hoveredSpan = computed(() => {
  const all = buckets.value;
  if (!hovered.value || !all?.length) return undefined;
  const [first, last] = hoverSpan(all.length, hovered.value);
  if (first >= last) return undefined;
  return { from: all[first].startTs, to: bucketEnd(all, last - 1, toNs.value), first, last };
});
const hover = computed(() => {
  const span = hoveredSpan.value;
  if (!span) return undefined;
  const count = buckets.value!.slice(span.first, span.last).reduce((sum, b) => sum + b.matched, 0);
  return { from: span.from, count, problem: problemAt(days.value, span.from, span.to) };
});

// A bar is a stretch of time, and the list below is the same time a day at a
// time: clicking one brings its row into view and marks it for a moment.
const highlighted = ref<string>();
let unmark: ReturnType<typeof setTimeout> | undefined;
function scrollToHovered() {
  const span = hoveredSpan.value;
  if (!span) return;
  const row = rowAt(days.value, span.from, span.to);
  if (!row) return;
  const rows = root.value?.querySelectorAll<HTMLElement>("[data-row]") ?? [];
  [...rows].find((el) => el.dataset.row === row.key)?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  highlighted.value = row.key;
  clearTimeout(unmark);
  unmark = setTimeout(() => (highlighted.value = undefined), 1500);
}
onBeforeUnmount(() => clearTimeout(unmark));

// The panel is open, so whatever it is showing has been looked at, and only
// that: the watermark moves to the newest alert on screen, never the
// instance's. An effect because the rows usually land after the panel opens.
// Origins only: a follow-up anchor carries the time of a later batch folded in
// silently, and marking that would put out dots for alerts elsewhere nobody saw.
// Nothing is marked while the panel shows a failure instead of rows.
const showing = computed(() => loaded.value && !failed.value);
watchEffect(() => {
  if (!showing.value) return;
  const newest = hits.value.reduce<CloudAlert | undefined>(
    (n, h) => (h.isOrigin && (!n || h.ts > n.ts) ? h : n),
    undefined,
  );
  if (newest) markAlertsSeen(newest);
});
// The dot counts recent alerts the timeline can leave out (a container since
// recreated, or older than the window), so opening the panel also clears what
// the dot counted in this scope; otherwise it stays lit with nothing to show.
watchEffect(() => {
  if (showing.value && dotted.value.length) markAlertsSeen(dotted.value[0]);
});
</script>
