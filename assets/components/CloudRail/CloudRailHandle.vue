<template>
  <!--
    What a hidden rail leaves behind: a tab on the edge, the way the nav leaves
    one on the left. Small enough to forget, findable when you want the
    assistant back.
  -->
  <button
    type="button"
    class="border-base-content/15 bg-base-200 hover:border-primary text-base-content/60 fixed top-1/2 right-0 z-30 -translate-y-1/2 rounded-l-md border border-r-0 py-3 pr-0.5 pl-1 shadow-sm transition-colors"
    :title="$t('cloud-rail.show')"
    :aria-label="$t('cloud-rail.show')"
    @click="showRail()"
  >
    <mdi:chevron-left class="size-4" />
  </button>
</template>

<script lang="ts" setup>
const { showRail } = useCloudRail();
</script>
