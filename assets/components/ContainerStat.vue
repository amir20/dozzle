<template>
  <div>
    {{ formatBytes(memoryUsage) }}
  </div>
</template>

<script>
import { mapGetters } from "vuex";

export default {
  props: {
    id: String,
  },
  name: "ContainerStat",
  computed: {
    ...mapGetters(["allContainersById"]),
    container() {
      return this.allContainersById[this.id];
    },
    stat() {
      return this.allContainersById[this.id] && this.allContainersById[this.id].stat
        ? this.allContainersById[this.id].stat
        : {};
    },
    memoryUsage() {
      return this.stat.memoryUsage;
    },
  },
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
.name {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(0, 0, 0, 0.1);
  font-weight: bold;
  font-family: monospace;

  button.delete {
    background-color: var(--scheme-main-ter);
    opacity: 0.6;
    &:after,
    &:before {
      background-color: var(--text-color);
    }

    &:hover {
      opacity: 1;
    }
  }
}
</style>
