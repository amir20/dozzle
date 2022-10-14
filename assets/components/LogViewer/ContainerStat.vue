<template>
  <div class="is-size-7 is-uppercase columns is-marginless is-mobile is-vcentered" v-if="container.stat">
    <div class="column is-narrow has-text-weight-bold">
      {{ container.state }}
    </div>
    <div class="column is-narrow has-text-centered is-relative">
      <div class="has-border">
        <stat-sparkline :data="memoryData"></stat-sparkline>
      </div>

      <div class="has-background-body-color is-top-left">
        <span class="has-text-weight-light has-spacer">mem</span>
        <span class="has-text-weight-bold">
          {{ formatBytes(container.stat.memoryUsage) }}
        </span>
      </div>
    </div>

    <div class="column is-narrow has-text-centered is-relative">
      <div class="has-border">
        <stat-sparkline :data="cpuData"></stat-sparkline>
      </div>
      <div class="has-background-body-color is-top-left">
        <span class="has-text-weight-light has-spacer">load</span>
        <span class="has-text-weight-bold"> {{ container.stat.cpu }}% </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const container = inject("container") as ComputedRef<Container>;

const cpuData = computedWithControl(
  () => container.value.getLastStat(),
  () => {
    const history = container.value.getStatHistory();
    return history.map((stat, i) => ({ x: history.length - i, y: stat.snapshot.cpu }));
  }
);

const memoryData = computedWithControl(
  () => container.value.getLastStat(),
  () => {
    const history = container.value.getStatHistory();
    return history.map((stat, i) => ({ x: history.length - i, y: stat.snapshot.memory }));
  }
);
</script>

<style lang="scss" scoped>
.has-spacer {
  &::after {
    content: " ";
  }
}

.has-border {
  border: 1px solid var(--primary-color);
  border-radius: 3px;
  padding: 1px 1px 0 1px;
  display: flex;
  overflow: hidden;
  padding-top: 0.25em;
}

.has-background-body-color {
  background-color: var(--body-background-color);
}

.is-top-left {
  position: absolute;
  top: 0;
  left: 0.75em;
}
</style>
