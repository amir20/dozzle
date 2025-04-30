<template>
  <nav class="border-base-content/20 bg-base-200 pt-safe fixed top-0 z-10 w-full border-b" data-testid="navigation">
    <div class="p-2">
      <div class="flex items-center">
        <router-link :to="{ name: '/' }">
          <Logo class="h-10" />
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
        <div v-show="show" class="flex h-[calc(100svh-55px)]">
          <SideMenu class="flex-1" />
        </div>
      </transition>
    </div>
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
<style scoped>
@reference "@/main.css";
.fade-enter-active,
.fade-leave-active {
  @apply transition-opacity;
}

.fade-enter-active > div,
.fade-leave-active > div {
  @apply transition-transform;
}

.fade-enter-from,
.fade-leave-to {
  @apply opacity-0;
}

.fade-enter-from > div,
.fade-leave-to > div {
  @apply -translate-y-10;
}
</style>
