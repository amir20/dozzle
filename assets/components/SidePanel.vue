<template>
  <aside class="fixed h-screen w-[inherit] overflow-auto p-3" data-testid="navigation">
    <h1>
      <router-link :to="{ name: '/' }">
        <svg class="h-14 w-28 fill-secondary">
          <use href="#logo"></use>
        </svg>
      </router-link>

      <small class="mb-4 block text-xs font-light" v-if="hostname">
        {{ hostname }}
      </small>
    </h1>

    <button
      class="input input-sm mt-4 inline-flex cursor-pointer items-center gap-2 font-light hover:border-primary"
      @click="$emit('search')"
      :title="$t('tooltip.search')"
      data-testid="search"
    >
      <mdi:magnify />
      {{ $t("placeholder.search") }}
      <key-shortcut char="k" class="text-base-content/70"></key-shortcut>
    </button>

    <side-menu class="mt-4"></side-menu>
  </aside>
</template>

<script lang="ts" setup>
const { hostname } = config;
</script>
