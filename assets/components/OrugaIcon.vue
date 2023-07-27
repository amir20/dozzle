<template>
  <div>
    <mdi:arrow-up v-if="icon[1] == 'arrow-up'" />
    <mdi:chevron-right v-else-if="icon[1] == 'chevron-right'" />
    <mdi:chevron-left v-else-if="icon[1] == 'chevron-left'" />
  </div>
</template>

<script lang="ts" setup>
const { icon } = defineProps<{ icon: string[] }>();
</script>

<style scoped></style>
