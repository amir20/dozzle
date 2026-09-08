<template>
  <!-- Deliberately unboxed: in a wall of monospace text the timestamp is a
       reference, not content, so it reads as a quiet left column and only
       sharpens when the row is hovered. -->
  <div
    class="text-base-content/45 group-hover/entry:text-base-content/80 pt-px whitespace-nowrap tabular-nums transition-colors"
  >
    <DateTime :date="date" />
  </div>
</template>
<script lang="ts" setup>
defineProps<{
  date: Date;
}>();
</script>
