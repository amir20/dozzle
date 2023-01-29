<template>
  <div class="is-size-7 is-uppercase columns is-marginless is-mobile is-vcentered" v-if="container.stat">
    <stat-monitor
      class="column is-narrow"
      :data="memoryData"
      label="mem"
      :stat-value="formatBytes(container.stat.memoryUsage)"
    ></stat-monitor>
    <stat-monitor
      class="column is-narrow"
      :data="cpuData"
      label="load"
      :stat-value="container.stat.cpu + '%'"
    ></stat-monitor>
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
    const points: Point<unknown>[] = history.map((stat, i) => ({
      x: i,
      y: stat.snapshot.cpu,
      value: stat.snapshot.cpu + "%",
    }));
    return points;
  }
);

const memoryData = computedWithControl(
  () => container.value.getLastStat(),
  () => {
    const history = container.value.getStatHistory();
    const points: Point<string>[] = history.map((stat, i) => ({
      x: i,
      y: stat.snapshot.memory,
      value: formatBytes(stat.snapshot.memoryUsage),
    }));
    return points;
  }
);
</script>

<style lang="scss" scoped>
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
