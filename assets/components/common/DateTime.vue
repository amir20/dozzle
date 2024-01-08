<template>
  <div class="inline-flex gap-2">
    <time :datetime="date.toISOString()" class="mobile-hidden">{{ dateStr }}</time>
    <time :datetime="date.toISOString()">{{ timeStr }}</time>
  </div>
</template>

<script lang="ts" setup>
const props = defineProps<{
  date: Date;
}>();

const dateOverride = computed(() => (dateLocale.value === "auto" ? undefined : dateLocale.value));
const dateFormatter = computed(
  () => new Intl.DateTimeFormat(dateOverride.value, { day: "2-digit", month: "2-digit", year: "numeric" }),
);
const use12Hour = computed(() => ({ auto: undefined, "12": true, "24": false })[hourStyle.value]);
const timeFormatter = computed(
  () =>
    new Intl.DateTimeFormat(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: use12Hour.value,
    }),
);

const dateStr = computed(() => dateFormatter.value.format(props.date));
const timeStr = computed(() => timeFormatter.value.format(props.date));
</script>
