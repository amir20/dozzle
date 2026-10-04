<template>
  <!-- A server page while the setup status is on its way, or could not be read. -->
  <div v-if="loading" class="flex justify-center py-6">
    <span class="loading loading-spinner loading-sm"></span>
  </div>
  <InlineNotice v-else type="error">{{ $t("setup.error.load") }}</InlineNotice>
</template>

<script lang="ts" setup>
defineProps<{ loading: boolean }>();
</script>
