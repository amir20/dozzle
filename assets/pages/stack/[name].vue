<template>
  <Search />
  <StackLog :name="name" />
</template>

<script lang="ts" setup>
const { name } = defineProps<{ name: string }>();
</script>
