<template>
  <time :datetime="date.toISOString()">{{ text }}</time>
</template>

<script lang="ts">
import formatDistance from "date-fns/formatDistance";

export default {
  props: {
    date: {
      required: true,
      type: Date,
    },
  },
  data() {
    return {
      text: "" as string,
      interval: null,
    };
  },
  name: "PastTime",
  mounted() {
    this.updateFromNow();
    this.interval = setInterval(() => this.updateFromNow(), 30000);
  },
  destroyed() {
    clearInterval(this.interval);
  },
  methods: {
    updateFromNow() {
      this.text = formatDistance(this.date, new Date(), {
        addSuffix: true,
      });
    },
  },
};
</script>

<style scoped lang="scss"></style>
