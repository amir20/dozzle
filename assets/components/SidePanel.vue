<template>
  <aside class="fixed flex h-screen w-[inherit] flex-col gap-4 p-3" data-testid="navigation">
    <h1>
      <router-link :to="{ name: '/' }">
        <LogoWithText class="logo h-16 w-40" />
      </router-link>

      <small class="mb-4 block text-xs font-light" v-if="hostname">
        {{ hostname }}
      </small>
    </h1>

    <button
      class="input input-sm inline-flex cursor-pointer items-center gap-2 self-start font-light hover:border-primary"
      @click="$emit('search')"
      :title="$t('tooltip.search')"
      data-testid="search"
    >
      <mdi:magnify />
      {{ $t("placeholder.search") }}
      <key-shortcut char="k" class="text-base-content/70"></key-shortcut>
    </button>

    <SideMenu class="mt-2 flex-1" />
  </aside>
</template>

<script lang="ts" setup>
import LogoWithText from "@/logo-text.svg";
const { hostname } = config;
</script>

<style scoped lang="postcss">
.logo {
  :deep(.content-fill) {
    @apply fill-base-content;
  }

  :deep(.secondary-fill) {
    @apply fill-secondary;
  }
}
</style>
