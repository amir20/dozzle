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

<script setup>
import { defineProps } from "vue";
defineProps({
  stat: Object,
  state: String,
});
function formatBytes(bytes, decimals = 2) {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}
</script>

<style lang="scss" scoped>
.has-spacer {
  &::after {
    content: " ";
  }
}
</style>
