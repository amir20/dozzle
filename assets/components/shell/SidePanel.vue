<template>
  <aside class="fixed flex h-screen w-[inherit] flex-col gap-3 p-3" data-testid="navigation">
    <!-- The wordmark used to run at text-4xl and ate the top of the sidebar. It is
         a brand mark, not a heading you read twice, so it gives the space back to
         the tree underneath. -->
    <h1 class="shrink-0">
      <router-link :to="{ name: '/' }" class="flex w-full items-center gap-2 overflow-hidden">
        <Logo class="size-9 shrink-0" aria-hidden="true" />
        <span class="truncate text-2xl font-light tracking-tight">Dozzle</span>
        <ProBadge />
      </router-link>

      <small class="text-base-content/45 mt-1.5 block truncate font-mono text-xs" v-if="hostname" :title="hostname">
        {{ hostname }}
      </small>
    </h1>

    <div class="bg-base-content/10 h-px shrink-0"></div>

    <SideMenu class="min-h-0 flex-1" />
  </aside>
</template>

<script lang="ts" setup>
import Logo from "@/logo.svg";
const { hostname } = config;
</script>

<style scoped></style>
