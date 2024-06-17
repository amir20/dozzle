<script setup>
import { data } from "../../activity.data.ts";
import Counter from "./Counter.vue";
</script>

<template>
  <div class="stats mt-10">
    <div class="stat w-40">
      <div class="stat-title">Github Stars</div>
      <div class="stat-value">
        <Counter :start="0" :end="data.stars" :duration="1000" :formatter="(value) => (value / 1e3).toFixed(1) + 'K'" />
      </div>
    </div>

    <div class="stat !border-l-0">
      <div class="stat-title">Docker Pulls</div>
      <div class="stat-value">
        <Counter :start="0" :end="data.pulls" :duration="1000" :formatter="(value) => (value / 1e6).toFixed(0) + 'M'" />
      </div>
    </div>
  </div>
</template>
