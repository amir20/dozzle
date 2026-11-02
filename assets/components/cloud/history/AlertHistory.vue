<template>
  <div>
    <!-- No heading of its own: the tab that opened this already names it.

      The panel keeps its shape across states: loading, empty, failed and full
      all render the same bordered surface, so nothing reshuffles underneath the
      reader when the fetch lands.
    -->
    <div
      v-if="(loading && !loaded) || failed || days.length === 0"
      class="border-base-content/15 bg-base-200/40 rounded-lg border"
    >
      <p v-if="loading && !loaded" class="text-base-content/60 p-4 text-sm">
        {{ $t("notifications.history.loading") }}
      </p>
      <p v-else-if="failed" class="text-base-content/60 p-4 text-sm">
        {{ $t("notifications.history.failed") }}
      </p>
      <p v-else class="text-base-content/60 p-4 text-sm">
        {{ $t("notifications.history.empty") }}
      </p>
    </div>

    <!-- One panel per day. A flat list of relative times made "2 days ago" and
         "3 days ago" read as the same blur; a day heading answers "what happened
         yesterday" without making the reader do the arithmetic. -->
    <div v-else class="space-y-6">
      <section v-for="day in days" :key="day.key">
        <h3 class="text-base-content/60 mb-2 flex items-baseline gap-2 text-xs font-semibold tracking-wide uppercase">
          {{ day.label }}
          <span class="text-base-content/40 font-mono font-normal">{{ day.items.length }}</span>
        </h3>
        <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
          <AlertRow v-for="alert in day.items" :key="alert.alertId" :alert="alert" />
        </div>
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { alerts, fetchRecentAlerts, refreshRecentAlerts, loading, loaded, failed } = useRecentAlerts();
const { locale } = useI18n();

// The rows are cached for the page, which is what keeps the bell and the dots on
// a container table to one request between them. This surface is the history
// itself, so opening it re-reads rather than showing what the bell happened to
// fetch a minute ago. Either branch is one request: fill the list if nothing has
// read it yet, re-read it if something has.
onMounted(() => (loaded.value ? refreshRecentAlerts() : fetchRecentAlerts()));

// Reads the shared tick so "Today" rolls over at midnight on a page left open.
const days = computed(() => {
  void relativeTimeTick.value;
  return groupByDay(alerts.value, (a) => new Date(a.ts / 1e6), locale.value);
});
</script>
