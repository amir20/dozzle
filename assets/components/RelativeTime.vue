<template>
  <time :datetime="date.toISOString()">{{ relativeTime(date, locale) }}</time>
</template>

<script lang="ts">
const use24Hr =
  new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
  })
    .formatToParts(new Date(2020, 0, 1, 13))
    .find((part) => part.type === "hour")?.value.length === 2;

const auto = use24Hr ? enGB : enUS;
const styles = { auto, 12: enUS, 24: enGB };
</script>

<script lang="ts" setup>
import { formatRelative } from "date-fns";
import enGB from "date-fns/locale/en-GB";
import enUS from "date-fns/locale/en-US";

defineProps<{
  date: Date;
}>();

const locale = computed(() => {
  const locale = styles[hourStyle.value];
  const oldFormatter = locale.formatRelative as (d: Date | number) => string;
  return {
    ...locale,
    formatRelative(date: Date | number) {
      return oldFormatter(date) + "p";
    },
  };
});

function relativeTime(date: Date, locale: Locale) {
  return formatRelative(date, new Date(), { locale });
}
</script>
