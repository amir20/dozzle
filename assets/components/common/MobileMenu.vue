<template>
  <nav class="fixed top-0 z-10 w-full border-b border-base-content/20 bg-base p-2" data-testid="navigation">
    <div class="flex items-center">
      <router-link :to="{ name: '/' }">
        <Logo class="logo h-10 w-14" />
      </router-link>

      <div class="ml-auto flex items-center gap-2">
        <a class="btn btn-circle flex" @click="$emit('search')" :title="$t('tooltip.search')">
          <mdi:magnify class="size-5" />
        </a>
        <label class="btn btn-circle swap swap-rotate" data-testid="hamburger">
          <input type="checkbox" v-model="show" />
          <mdi:close class="swap-on" />
          <mdi:hamburger-menu class="swap-off" />
        </label>
      </div>
    </div>

    <transition name="fade">
      <div v-show="show" class="h-[calc(100vh-60px)] overflow-auto">
        <SideMenu />
      </div>
    </transition>
  </nav>
</template>

<script lang="ts" setup>
import Logo from "@/logo.svg";
const route = useRoute();

const show = ref(false);
watch(route, () => {
  show.value = false;
});
</script>
<style scoped lang="postcss">
li.exited {
  @apply opacity-50;
}

.fade-enter-active,
.fade-leave-active {
  @apply transition-opacity;
}

.fade-enter-active .menu,
.fade-leave-active .menu {
  @apply transition-transform;
}

.fade-enter-from,
.fade-leave-to {
  @apply opacity-0;
}

.fade-enter-from .menu,
.fade-leave-to .menu {
  @apply -translate-y-2;
}

.logo {
  :deep(.secondary-fill) {
    @apply fill-secondary;
  }
}
</style>
