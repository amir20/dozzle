<template>
  <!-- A phone's header. On a desktop the same entries sit at the foot of the right
       rail, so this stays mounted (it owns the alerts polling) but is not shown. -->
  <div class="flex items-center justify-end gap-4" :class="{ hidden: !isMobile }">
    <slot name="more-items"></slot>
    <Announcements />

    <UserMenu />

    <CloudLinkPopover v-if="canLinkCloud && !cloudLinked" />
    <!-- A phone has no rail, so once linked this is how Cloud is reached from any
         page. It opens on alerts when one fired unseen, else where the reader left off. -->
    <button
      v-else-if="isMobile && railAvailable"
      type="button"
      class="btn btn-circle btn-sm relative"
      :aria-label="$t('cloud.title')"
      data-testid="cloud-open"
      @click="openCloud(unseenAlerts)"
    >
      <mdi:cloud class="icon-float text-info size-6" />
      <span v-if="unseenAlerts" class="bg-warning absolute end-1 top-1 size-1.5 rounded-full"></span>
    </button>
  </div>
</template>
<script lang="ts" setup>
const { canLink: canLinkCloud, linked: cloudLinked } = useCloudSurface();

// The header owns the polling that keeps the rail's and the mobile sheet's
// unseen dot honest on the pages that mount it (home, notifications, settings,
// search). The log views do not, so the dot there is as fresh as the last of
// those pages. Without a cloud link there is nothing to remember and nothing to poll.
const { unseen: unseenAlerts, fetchRecentAlerts, refreshRecentAlerts } = useRecentAlerts();
const { available: railAvailable, openCloud } = useCloudRail();

onMounted(() => fetchRecentAlerts());

// A minute is slow enough to be invisible and fast enough that the dot is not a
// lie. Paused while the tab is hidden, and caught up the moment it comes back.
const visibility = useDocumentVisibility();
const { pause, resume } = useIntervalFn(() => refreshRecentAlerts(), 60_000);
watch(visibility, (state) => {
  if (state === "visible") {
    refreshRecentAlerts();
    resume();
  } else {
    pause();
  }
});
</script>
