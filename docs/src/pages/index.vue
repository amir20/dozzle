<script setup lang="ts">
defineOptions({
  name: 'IndexPage',
})
</script>

<template>
  <hero />
  <div mx-auto container space-y-14 divide-y divide="gray-200 dark:gray-800">
    <features />
    <installation />
  </div>
</template>
