<template>
  <time :datetime="date.toISOString()" data-ci-skip>{{ text }}</time>
</template>

<script lang="ts" setup>
const {
  date,
  strict = false,
  suffix = true,
} = defineProps<{
  date: Date;
  strict?: boolean;
  suffix?: boolean;
}>();

const text = ref<string>();

watch($$(date), updateFromNow, { immediate: true });

function updateFromNow() {
  text.value = getRelativeTime(date, locale.value === "" ? undefined : locale.value);
}
useIntervalFn(updateFromNow, 30_000, { immediateCallback: true });

function getRelativeTime(date: Date, locale: string | undefined): string {
  const diffInSeconds = (date.getTime() - new Date().getTime()) / 1000;
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });

  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 31536000],
    ["month", 2592000],
    ["week", 604800],
    ["day", 86400],
    ["hour", 3600],
    ["minute", 60],
    ["second", 1],
  ];

  for (const [unit, seconds] of units) {
    const value = Math.round(diffInSeconds / seconds);
    if (Math.abs(value) >= 1) {
      return rtf.format(value, unit);
    }
  }

  return rtf.format(0, "second");
}
</script>
