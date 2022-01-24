<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <span class="column is-ellipsis">
      {{ container.name }}
      <span class="tag is-dark">{{ container.image.replace(/@sha.*/, "") }}</span>
    </span>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/types/Container";
import { PropType } from "vue";
defineProps({
  container: {
    type: Object as PropType<Container>,
    required: true,
  },
});
</script>

<style lang="scss" scoped></style>
