<template>
  <!--
    What fired, scoped to the containers on screen or, widened or on a page
    with none, to the whole instance. This is the alert history: there is no
    second copy of it on the notifications page.

    One panel per day. A flat list of relative times made "2 days ago" and
    "3 days ago" read as the same blur; a day heading answers "what happened
    yesterday" without making the reader do the arithmetic.
  -->
  <div class="h-full overflow-y-auto p-4">
    <p v-if="loading && !loaded" class="text-base-content/60 text-sm">
      {{ $t("notifications.history.loading") }}
    </p>
    <p v-else-if="failed" class="text-base-content/60 text-sm">
      {{ $t("notifications.history.failed") }}
    </p>
    <!-- Nothing fired: say how far back that answer reaches, and where a rule
         that would fire is made. -->
    <div v-else-if="!visible.length" class="space-y-2 text-sm" data-empty>
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

    <div v-else class="space-y-6">
      <section v-for="day in days" :key="day.key">
        <h3 class="text-base-content/60 mb-2 flex items-baseline gap-2 text-xs font-semibold tracking-wide uppercase">
          {{ day.label }}
          <span class="text-base-content/40 font-mono font-normal">{{ day.items.length }}</span>
        </h3>
        <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <AlertRow v-for="alert in day.items" :key="alert.alertId" :alert="alert" compact :hide-container="single" />
        </div>
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { fetchRecentAlerts, refreshRecentAlerts, markAlertsSeen, loading, loaded, failed } = useRecentAlerts();
const { visible, scoped, single } = useViewAlerts();
const { locale } = useI18n();
const { closeRail } = useCloudRail();
const { cloudStatus, ensureCloudStatus } = useCloudConfig();
const retentionDays = computed(() => cloudStatus.value?.plan.retention_days ?? 0);
onMounted(ensureCloudStatus);

// The rows are cached for the page, which keeps the bell and the dots on a
// container table to one request between them. Opening the history re-reads
// rather than showing what the bell happened to fetch a minute ago.
onMounted(() => (loaded.value ? refreshRecentAlerts() : fetchRecentAlerts()));

// Reads the shared tick so "Today" rolls over at midnight on a page left open.
const days = computed(() => {
  void relativeTimeTick.value;
  return groupByDay(visible.value, (a) => new Date(a.ts / 1e6), locale.value);
});

// The panel is open, so whatever it is showing has been looked at, and only
// that: marking the instance's newest from a panel scoped to one container put
// out the nav's bell for alerts that were never on this screen. An effect
// because the rows usually land after the panel opens.
watchEffect(() => {
  if (visible.value.length) markAlertsSeen(visible.value[0]);
});
</script>
