<template>
  <search></search>
  <log-stack :name="name"></log-stack>
</template>

<script lang="ts" setup>
const { name } = defineProps<{ name: string }>();
</script>
