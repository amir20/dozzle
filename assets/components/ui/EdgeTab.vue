<template>
  <!--
    What a hidden column leaves behind: a slim tab stuck to the screen edge, vertically
    centered. Small enough to forget, findable when you want the column back. The nav
    and the cloud rail both use it, one per edge, so the two read as the same gesture.
  -->
  <button
    type="button"
    class="border-base-content/15 bg-base-200 hover:bg-base-300 hover:text-base-content text-base-content/60 fixed top-1/2 z-30 -translate-y-1/2 border py-3 shadow-sm transition-colors"
    :class="
      side === 'left' ? 'left-0 rounded-r-md border-l-0 pr-1 pl-0.5' : 'right-0 rounded-l-md border-r-0 pr-0.5 pl-1'
    "
    :title="label"
    :aria-label="label"
  >
    <mdi:chevron-right v-if="side === 'left'" class="size-4" />
    <mdi:chevron-left v-else class="size-4" />
  </button>
</template>

<script lang="ts" setup>
defineProps<{ side: "left" | "right"; label: string }>();
</script>
