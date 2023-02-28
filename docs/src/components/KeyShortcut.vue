<script setup lang="ts">
const { label } = $defineProps<{
  label: string;
}>();
</script>

<template>
  <span border="rounded-lg light-50" bg="light-200" text="dark-200" p-1> ⌘ {{ label }} </span>
</template>
