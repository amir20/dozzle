<template>
  <aside class="fixed flex h-screen w-[inherit] flex-col gap-4 p-3" data-testid="navigation">
    <h1>
      <router-link :to="{ name: '/' }" class="flex w-full items-center gap-4 overflow-hidden text-4xl font-thin">
        <Logo class="h-14 w-14 shrink-0" />
        Dozzle
      </router-link>

      <small class="mt-4 block text-sm font-light" v-if="hostname">{{ hostname }}</small>
    </h1>

    <button
      class="input input-sm hover:border-primary mt-2 inline-flex w-auto cursor-pointer items-center gap-2 self-start font-light"
      @click="$emit('search')"
      :title="$t('tooltip.search')"
      data-testid="search"
    >
      <mdi:magnify />
      {{ $t("placeholder.search") }}
      <key-shortcut char="k" class="text-base-content/70"></key-shortcut>
    </button>

    <SideMenu class="flex-1" />
  </aside>
</template>

<script lang="ts" setup>
import Logo from "@/logo.svg";
const { hostname } = config;
</script>

<style scoped></style>
