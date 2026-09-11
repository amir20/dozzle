<template>
  <!--
    What fired on what you are looking at.

    The notifications page answers the same question for the whole instance;
    this one is scoped to the containers in view, which is the only reason it
    earns a place beside the stream rather than a link to that page.
  -->
  <div class="h-full overflow-y-auto">
    <p v-if="loading && !loaded" class="text-base-content/60 p-4 text-sm">
      {{ $t("notifications.history.loading") }}
    </p>
    <p v-else-if="failed" class="text-base-content/60 p-4 text-sm">
      {{ $t("notifications.history.failed") }}
    </p>
    <p v-else-if="!visible.length" class="text-base-content/60 p-4 text-sm">
      {{ scoped ? $t("cloud-rail.alerts-empty-here") : $t("notifications.history.empty") }}
    </p>

    <div v-else class="divide-base-content/10 divide-y">
      <AlertRow v-for="alert in visible" :key="alert.alertId" :alert="alert" compact :hide-container="single" />
    </div>

    <!-- The rest of the history is a page, not a panel: this one is about the
         view, and everything else belongs where it can be filtered and read. -->
    <div v-if="visible.length" class="border-base-content/10 border-t p-3">
      <RouterLink to="/notifications" class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm">
        <mdi:history class="size-4 opacity-60" />
        {{ $t("cloud-rail.all-activity") }}
      </RouterLink>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { fetchRecentAlerts, markAlertsSeen, loading, loaded, failed } = useRecentAlerts();
const { visible, scoped, single } = useViewAlerts();

onMounted(() => fetchRecentAlerts());

// The panel is open, so whatever it is showing has been looked at — and only
// that. Marking the instance's newest seen from a panel scoped to one container
// put out the nav's bell for alerts that were never on this screen, which is
// also why a panel showing nothing marks nothing.
//
// An effect rather than a one-shot in onMounted: the rows usually land after the
// panel opens, and more can arrive while it stays open.
watchEffect(() => {
  if (visible.value.length) markAlertsSeen(visible.value[0]);
});
</script>
