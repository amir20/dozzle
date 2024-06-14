<script setup>
import { data } from "../../starCounter.data.ts";
import Counter from "./Counter.vue";
</script>

<template>
  <Counter :start="0" :end="data.stars" :duration="2000" />
</template>
