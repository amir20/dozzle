<template>
  <EdgeTab side="right" :label="$t('cloud-rail.show')" @click="showRail()" />
</template>

<script lang="ts" setup>
const { showRail } = useCloudRail();
</script>
