<template>
  <div>
    <!-- No heading of its own: the tab that opened this already names it.

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

      <!-- Part of the chain above, not a sibling of it. As a sibling the rows
           rendered underneath whichever line was showing, so unlinking in the
           settings card printed "linking would remember these" over a list of
           remembered alerts. -->
      <template v-else>
        <AlertRow v-for="alert in alerts" :key="alert.alertId" :alert="alert" />
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { alerts, fetchRecentAlerts, loading, loaded, failed } = useRecentAlerts();
const { linked } = useCloudSurface();

onMounted(() => fetchRecentAlerts());
</script>
