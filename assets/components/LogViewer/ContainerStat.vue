<template>
  <div class="is-size-7 is-uppercase columns is-marginless is-mobile is-vcentered" v-if="container.stat">
    <div class="column is-narrow has-text-weight-bold">
      {{ container.state }}
    </div>
    <div class="column is-narrow">
      <span class="has-text-weight-light has-spacer">mem</span>
      <span class="has-text-weight-bold">
        {{ formatBytes(container.stat.memoryUsage) }}
      </span>
    </div>

    <div class="column is-narrow">
      <span class="has-text-weight-light has-spacer">load</span>
      <span class="has-text-weight-bold"> {{ container.stat.cpu }}% </span>
    </div>
    <div class="column is-narrow">
      <cpu-sparkline></cpu-sparkline>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const container = inject("container") as ComputedRef<Container>;
</script>

<style lang="scss" scoped>
.has-spacer {
  &::after {
    content: " ";
  }
}
</style>
