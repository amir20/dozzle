<template>
  <!-- A value a flag or environment variable fixes, named so the operator knows where to change it. -->
  <span class="text-base-content/40 flex items-center gap-1 text-xs">
    <mdi:lock-outline class="size-3.5 shrink-0" />
    {{ $t("setup.actions.locked", { env }) }}
  </span>
</template>

<script lang="ts" setup>
defineProps<{ env: string }>();
</script>
