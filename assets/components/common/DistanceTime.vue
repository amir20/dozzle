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
  console.log("Locale:", locale);
  const now = new Date();
  const diffInSeconds = (date.getTime() - now.getTime()) / 1000;

  const units: { unit: Intl.RelativeTimeFormatUnit; seconds: number }[] = [
    { unit: "year", seconds: 31536000 },
    { unit: "month", seconds: 2592000 },
    { unit: "week", seconds: 604800 },
    { unit: "day", seconds: 86400 },
    { unit: "hour", seconds: 3600 },
    { unit: "minute", seconds: 60 },
    { unit: "second", seconds: 1 },
  ];

  for (const { unit, seconds } of units) {
    const value = Math.round(diffInSeconds / seconds);
    if (Math.abs(value) >= 1) {
      const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
      return rtf.format(value, unit);
    }
  }

  return new Intl.RelativeTimeFormat(locale, { numeric: "auto" }).format(0, "second");
}
</script>
