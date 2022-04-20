<template>
  <div class="is-size-7 is-uppercase columns is-marginless is-mobile">
    <div class="column is-narrow has-text-weight-bold">
      {{ state }}
    </div>
    <div class="column is-narrow" v-if="stat.memoryUsage !== null">
      <span class="has-text-weight-light has-spacer">mem</span>
      <span class="has-text-weight-bold">
        {{ formatBytes(stat.memoryUsage) }}
      </span>
    </div>

    <div class="column is-narrow" v-if="stat.cpu !== null">
      <span class="has-text-weight-light has-spacer">load</span>
      <span class="has-text-weight-bold"> {{ stat.cpu }}% </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ContainerStat } from "@/types/Container";
import { PropType } from "vue";
import { formatBytes } from "@/utils";

defineProps({
  stat: {
    type: Object as PropType<ContainerStat>,
    required: true,
  },
  state: String,
});
</script>

<style lang="scss" scoped>
.has-spacer {
  &::after {
    content: " ";
  }
}
</style>
