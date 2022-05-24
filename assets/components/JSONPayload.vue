<template>
  <span class="text">
    <div v-for="(value, name, index) in payload">{{ index }}. {{ name }}: {{ value }}</div>
  </span>
</template>
<script lang="ts" setup>
import { PropType } from "vue";

defineProps({
  payload: {
    type: Object as PropType<object>,
    required: true,
  },
});
</script>
