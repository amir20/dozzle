<template>
  <time :datetime="date.toISOString()">{{ text }}</time>
</template>

<script lang="ts" setup>
import formatDistance from "date-fns/formatDistance";

const { date } = defineProps<{
  date: Date;
}>();

const text = ref<string>();
function updateFromNow() {
  text.value = formatDistance(date, new Date(), {
    addSuffix: true,
  });
}
useIntervalFn(updateFromNow, 30_000, { immediateCallback: true });
</script>
