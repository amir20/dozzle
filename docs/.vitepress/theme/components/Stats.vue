<template>
  <div class="stats stats-horizontal mt-10">
    <a class="stat w-40 border-0! no-underline!" href="https://github.com/amir20/dozzle/stargazers">
      <div class="stat-title text-(--vp-c-text-1)">Github Stars</div>
      <div class="stat-value">
        <Counter :start="0" :end="data.stars" :duration="1000" :formatter="compact" />
      </div>
    </a>

    <a class="stat border-0! no-underline!" href="https://hub.docker.com/r/amir20/dozzle">
      <div class="stat-title text-(--vp-c-text-1)">Docker Pulls</div>
      <div class="stat-value">
        <Counter :start="0" :end="data.pulls" :duration="1000" :formatter="compact" />
      </div>
    </a>
  </div>
</template>
<script lang="ts" setup>
import { data } from "../../activity.data";
import Counter from "./Counter.vue";

const compact = (value: number) => {
  if (value >= 1e9) return (value / 1e9).toFixed(1) + "B";
  if (value >= 1e6) return Math.round(value / 1e6) + "M";
  if (value >= 1e3) return (value / 1e3).toFixed(1) + "K";
  return Math.round(value).toString();
};
</script>
