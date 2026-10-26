<template>
  <!-- An image that declares no version shows nothing; the update itself is
       still listed wherever this sits. -->
  <span v-if="result?.latestVersion" class="inline-flex items-center gap-1.5 font-mono text-xs">
    <template v-if="result.currentVersion">
      <span class="text-base-content/40">{{ result.currentVersion }}</span>
      <span class="text-base-content/40">&rarr;</span>
    </template>
    <span class="font-semibold">{{ result.latestVersion }}</span>
  </span>
</template>

<script lang="ts" setup>
defineProps<{ result?: Pick<ImageUpdateResult, "currentVersion" | "latestVersion"> }>();
</script>
