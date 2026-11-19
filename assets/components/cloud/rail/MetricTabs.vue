<template>
  <!-- Which metric a ranked list is ordered by. -->
  <div role="tablist" class="tabs tabs-box tabs-xs">
    <button
      v-for="option in options"
      :key="option.value"
      role="tab"
      class="tab"
      :class="{ 'tab-active': model === option.value }"
      :aria-selected="model === option.value"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<script lang="ts" setup>
import type { Metric } from "@/composable/cloud/hostMetrics";

const model = defineModel<Metric>({ required: true });
const { metrics } = defineProps<{ metrics: Metric[] }>();
const { t } = useI18n();
const labels: Record<Metric, string> = {
  cpu: "label.cpu",
  memory: "label.mem",
  network: "label.network",
  disk: "label.disk-io",
};
const options = computed(() => metrics.map((value) => ({ value, label: t(labels[value]) })));
</script>
