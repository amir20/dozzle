<template>
  <time :datetime="date.toISOString()" data-ci-skip>{{ text }}</time>
</template>

<script lang="ts" setup>
import formatDistanceToNow from "date-fns/formatDistanceToNow";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

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
function updateFromNow() {
  const fn = strict ? formatDistanceToNowStrict : formatDistanceToNow;
  text.value = fn(date, {
    addSuffix: suffix,
  });
}
useIntervalFn(updateFromNow, 30_000, { immediateCallback: true });
</script>
