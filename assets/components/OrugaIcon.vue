<template>
  <div>
    <mdi:arrow-up v-if="icon[1] == 'arrow-up'" />
  </div>
</template>

<script lang="ts" setup>
const { icon } = defineProps<{ icon: string[] }>();
</script>

<style scoped></style>
