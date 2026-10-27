<template>
  <!-- An image that declares no version says why; the update itself is still
       listed wherever this sits. -->
  <span v-if="result?.latestVersion" class="inline-flex items-center gap-1.5 font-mono text-xs">
    <template v-if="result.currentVersion === result.latestVersion">
      <span class="font-semibold">{{ result.latestVersion }}</span>
      <span class="text-base-content/40">({{ $t("updates.version.rebuilt") }})</span>
    </template>
    <template v-else>
      <template v-if="result.currentVersion">
        <span class="text-base-content/40">{{ result.currentVersion }}</span>
        <span class="text-base-content/40">&rarr;</span>
      </template>
      <span class="font-semibold">{{ result.latestVersion }}</span>
    </template>
  </span>
  <span v-else-if="result?.versionReason" class="text-base-content/40 text-xs">
    {{ $t("updates.version.unknown", { reason: $t(`updates.version.${result.versionReason}`) }) }}
  </span>
</template>

<script lang="ts" setup>
defineProps<{ result?: Pick<ImageUpdateResult, "currentVersion" | "latestVersion" | "versionReason"> }>();
</script>
