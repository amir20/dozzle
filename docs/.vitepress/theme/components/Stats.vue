<template>
  <div class="stats stats-horizontal mt-10">
    <div class="stat w-40 border-0!">
      <div class="stat-title text-(--vp-c-text-1)">Github Stars</div>
      <div class="stat-value">
        <Counter :start="0" :end="data.stars" :duration="1000" :formatter="(value) => (value / 1e3).toFixed(1) + 'K'" />
      </div>
    </div>

    <div class="stat">
      <div class="stat-title text-(--vp-c-text-1)">Docker Pulls</div>
      <div class="stat-value">
        <Counter :start="0" :end="data.pulls" :duration="1000" :formatter="(value) => (value / 1e6).toFixed(0) + 'M'" />
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { data } from "../../activity.data";
import Counter from "./Counter.vue";
</script>
