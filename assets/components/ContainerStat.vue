<template>
  <div class="has-text-weight-light is-size-7 is-uppercase columns is-marginless">
    <div class="column is-narrow">
      {{ state }}
    </div>
    <div class="column is-narrow" v-if="stat.memoryUsage !== null">mem {{ formatBytes(stat.memoryUsage) }}</div>
    <div class="column is-narrow" v-if="stat.cpu !== null">load {{ stat.cpu }}%</div>
  </div>
</template>

<script>
export default {
  props: {
    stat: Object,
    state: String,
  },
  name: "ContainerStat",
  methods: {
    formatBytes(bytes, decimals = 2) {
      if (bytes === 0) return "0 Bytes";
      const k = 1024;
      const dm = decimals < 0 ? 0 : decimals;
      const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
    },
  },
};
</script>

<style lang="scss" scoped>
.column {
  padding-top: 0;
}
</style>
