<template>
  <time :datetime="date.toISOString()">{{ text }}</time>
</template>

<script lang="ts" setup>
import { useIntervalFn } from "@vueuse/core";
import formatDistance from "date-fns/formatDistance";
import { PropType, ref } from "vue";

const props = defineProps({
  date: {
    required: true,
    type: Object as PropType<Date>,
  },
});

const text = ref<string>();
function updateFromNow() {
  text.value = formatDistance(props.date, new Date(), {
    addSuffix: true,
  });
}
useIntervalFn(updateFromNow, 30_000, { immediateCallback: true });
</script>
