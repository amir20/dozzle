<template>
  <time :datetime="date.toISOString()">{{ text }}</time>
</template>

<script lang="ts" setup>
const { date, narrow = false } = defineProps<{
  date: Date;
  // "5m ago" instead of "5 minutes ago", for a column too tight for the long form.
  narrow?: boolean;
}>();

// Reading the shared tick is what re-evaluates this on the half minute. A timer per
// component put one setInterval behind every row of the container table.
const text = computed(() => {
  relativeTimeTick.value;
  return toRelativeTime(date, locale.value === "" ? undefined : locale.value, narrow ? "narrow" : "long");
});
</script>
