<script setup>
import { data } from "../../activity.data.ts";
import Counter from "./Counter.vue";
</script>

<template>
  <div class="flex flex-row gap-4 pt-10">
    <div class="w-35 flex items-center rounded-md border-solid border-white p-4 text-center">
      <i class="i-grommet-icons:star w-2em h-2em inline-block" />
      <Counter :start="0" :end="data.stars" :duration="1000" />
    </div>

    <div class="w-35 flex items-center rounded-md border-solid border-white p-4 text-center">
      <Counter :start="0" :end="data.pulls" :duration="1000" />
    </div>
  </div>
</template>
