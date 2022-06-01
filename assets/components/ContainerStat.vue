<template>
  <div class="is-size-7 is-uppercase columns is-marginless is-mobile" v-if="container.stat">
    <div class="column is-narrow has-text-weight-bold">
      {{ container.state }}
    </div>
    <div class="column is-narrow" v-if="container.stat.memoryUsage !== null">
      <span class="has-text-weight-light has-spacer">mem</span>
      <span class="has-text-weight-bold">
        {{ formatBytes(container.stat.memoryUsage) }}
      </span>
    </div>

    <div class="column is-narrow" v-if="container.stat.cpu !== null">
      <span class="has-text-weight-light has-spacer">load</span>
      <span class="has-text-weight-bold"> {{ container.stat.cpu }}% </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/types/Container";
import { ComputedRef, inject } from "vue";
import { formatBytes } from "@/utils";

const container = inject("container") as ComputedRef<Container>;
</script>

<style lang="scss" scoped>
.has-spacer {
  &::after {
    content: " ";
  }
}
</style>
