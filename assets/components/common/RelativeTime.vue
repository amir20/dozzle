<template>
  <time :datetime="date.toISOString()" data-ci-skip>{{ text }}</time>
</template>

<script lang="ts" setup>
const { date } = defineProps<{
  date: Date;
}>();

const text = ref<string>();

const updateFromNow = () => {
  text.value = toRelativeTime(date, locale.value === "" ? undefined : locale.value);
};
watch(() => date, updateFromNow, { immediate: true });
useIntervalFn(updateFromNow, 30_000);
</script>
