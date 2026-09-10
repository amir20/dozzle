<template>
  <time :datetime="date.toISOString()">{{ text }}</time>
</template>

<script lang="ts" setup>
const { date } = defineProps<{
  date: Date;
}>();

// Reading the shared tick is what re-evaluates this on the half minute. A timer per
// component put one setInterval behind every row of the container table.
const text = computed(() => {
  relativeTimeTick.value;
  return toRelativeTime(date, locale.value === "" ? undefined : locale.value);
});
</script>
