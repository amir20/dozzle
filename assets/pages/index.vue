<template>
  <div class="section tile is-ancestor">
    <div class="tile is-parent">
      <div class="tile is-child box">
        <div class="level-item has-text-centered">
          <div>
            <p class="title">{{ runningContainers.length }} / {{ containers.length }}</p>
            <p class="heading">{{ $t("label.running") }} / {{ $t("label.total-containers") }}</p>
          </div>
        </div>
      </div>
    </div>
    <div class="tile is-parent">
      <div class="tile is-child box">
        <div class="level-item has-text-centered">
          <div>
            <p class="title">{{ totalCpu }}%</p>
            <p class="heading">{{ $t("label.total-cpu-usage") }}</p>
          </div>
        </div>
      </div>
    </div>
    <div class="tile is-parent">
      <div class="tile is-child box">
        <div class="level-item has-text-centered">
          <div>
            <p class="title">{{ formatBytes(totalMem) }}</p>
            <p class="heading">{{ $t("label.total-mem-usage") }}</p>
          </div>
        </div>
      </div>
    </div>
    <div class="tile is-parent">
      <div class="tile is-child box">
        <div class="level-item has-text-centered">
          <div>
            <p class="title">{{ version }}</p>
            <p class="heading">{{ $t("label.dozzle-version") }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>

  <section class="section table-container">
    <div class="box">
      <container-table :containers="runningContainers"></container-table>
    </div>
  </section>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { t } = useI18n();
const { version } = config;
const containerStore = useContainerStore();
const { containers, ready } = storeToRefs(containerStore) as unknown as {
  containers: Ref<Container[]>;
  ready: Ref<boolean>;
};

const mostRecentContainers = $computed(() => [...containers.value].sort((a, b) => +b.created - +a.created));
const runningContainers = $computed(() => mostRecentContainers.filter((c) => c.state === "running"));

let totalCpu = $ref(0);
useIntervalFn(
  () => {
    totalCpu = runningContainers.reduce((acc, c) => acc + c.stat.cpu, 0);
  },
  1000,
  { immediate: true },
);

let totalMem = $ref(0);
useIntervalFn(
  () => {
    totalMem = runningContainers.reduce((acc, c) => acc + c.stat.memoryUsage, 0);
  },
  1000,
  { immediate: true },
);

watchEffect(() => {
  if (ready.value) {
    setTitle(t("title.dashboard", { count: runningContainers.length }));
  }
});
</script>
<style lang="scss" scoped>
.panel {
  border: 1px solid var(--border-color);

  .panel-block,
  .panel-tabs {
    border-color: var(--border-color);

    .is-active {
      border-color: var(--border-hover-color);
    }

    .name {
      text-overflow: ellipsis;
      white-space: nowrap;
      overflow: hidden;
    }

    .status {
      margin-left: auto;
      white-space: nowrap;
    }
  }
}

@media screen and (max-width: 768px) {
  .pb-0-is-mobile {
    padding-bottom: 0 !important;
  }

  .pt-0-is-mobile {
    padding-top: 0 !important;
  }
}

.icon {
  padding: 10px 3px;
}

.bar-chart {
  height: 1.5em;
  .bar-text {
    font-size: 0.9em;
    padding: 0 0.5em;
  }
}

:deep(tr td) {
  padding-top: 1em;
  padding-bottom: 1em;
}

.section + .section {
  padding-top: 0;
}
</style>
