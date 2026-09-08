<template>
  <NavItem :label="host.name" :title="host.name" :muted="!host.available">
    <template #icon><HostIcon :type="host.type" class="size-4" /></template>
    <template #trailing v-if="!host.available">
      <span class="status-pill status-pill-error px-1.5 py-0 text-[0.6rem]">offline</span>
    </template>
  </NavItem>
</template>

<script lang="ts" setup>
import type { Host } from "@/stores/hosts";

const { host } = defineProps<{ host: Host }>();
</script>
