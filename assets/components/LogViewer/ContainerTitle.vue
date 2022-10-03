<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <span class="column is-ellipsis">
      {{ container.name }}
      <span class="tag is-dark">{{ container.image.replace(/@sha.*/, "") }}</span>
    </span>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const container = inject("container") as ComputedRef<Container>;
</script>

<style lang="scss" scoped></style>
