<template>
  <aside class="fixed flex h-screen w-[inherit] flex-col gap-4 p-3" data-testid="navigation">
    <h1>
      <router-link :to="{ name: '/' }" class="flex w-full items-center gap-4 overflow-hidden text-4xl font-thin">
        <Logo class="h-14 w-14 shrink-0" />
        Dozzle
      </router-link>

      <small class="mt-4 block text-sm font-light" v-if="hostname">{{ hostname }}</small>
    </h1>

    <SideMenu class="flex-1" />
  </aside>
</template>

<script lang="ts" setup>
import Logo from "@/logo.svg";
const { hostname } = config;
</script>

<style scoped></style>
