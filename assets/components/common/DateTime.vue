<template>
  <div class="inline-flex gap-2">
    <time :datetime="date.toISOString()" class="max-md:hidden">{{ dateStr }}</time>
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
const hourCycle = computed(() => {
  switch (hourStyle.value) {
    case "auto":
      return undefined;
    case "12":
      return "h12";
    case "24":
      return "h23";
  }
});
const timeFormatter = computed(
  () =>
    new Intl.DateTimeFormat(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hourCycle: hourCycle.value,
    }),
);

const dateStr = computed(() => dateFormatter.value.format(props.date));
const timeStr = computed(() => timeFormatter.value.format(props.date));
</script>
