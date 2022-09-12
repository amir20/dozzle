<template>
  <time :datetime="date.toISOString()">{{ format(date) }}</time>
</template>

<script lang="ts" setup>
defineProps<{
  date: Date;
}>();

// hourStyle
const dateFormatter = new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "2-digit", year: "numeric" });
const use12Hour = $computed(() => ({ auto: undefined, "12": true, "24": false }[hourStyle.value]));
const timeFormatter = $computed(
  () => new Intl.DateTimeFormat(undefined, { hour: "numeric", minute: "2-digit", second: "2-digit", hour12: use12Hour })
);

function format(date: Date) {
  const dateStr = dateFormatter.format(date);
  const timeStr = timeFormatter.format(date);
  return `${dateStr} ${timeStr}`;
}
</script>
