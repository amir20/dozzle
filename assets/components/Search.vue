<template>
  <transition name="slide">
    <div
      class="fixed z-10 flex w-full justify-end p-2"
      v-show="showSearch"
      v-if="search"
      ref="container"
      :style="style"
    >
      <div class="input input-primary flex items-center shadow-lg" :class="!isValidQuery ? 'input-warning' : ''">
        <mdi:magnify />
        <input
          class="input input-ghost w-72 flex-1"
          type="text"
          placeholder="Find / RegEx"
          ref="input"
          v-model="searchQueryFilter"
          @keyup.esc="resetSearch()"
        />
        <a class="btn btn-circle btn-xs" @click="resetSearch()"> <mdi:close /></a>
      </div>
    </div>
  </transition>
</template>

<script lang="ts" setup>
const input = ref<HTMLInputElement>();
const container = ref<HTMLDivElement>();
const { searchQueryFilter, showSearch, resetSearch, isValidQuery } = useSearchFilter();

const { style } = useDraggable(container);

onKeyStroke("f", (e) => {
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey) {
    showSearch.value = true;
    nextTick(() => input.value?.focus() || input.value?.select());
    e.preventDefault();
  }
});

onMounted(() => {
  onKeyStroke(
    "f",
    (e) => {
      if (e.ctrlKey || e.metaKey) {
        e.stopPropagation();
        resetSearch();
      }
    },
    { target: input.value },
  );
});

onUnmounted(() => resetSearch());
</script>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: all 200ms cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.slide-enter-from,
.slide-leave-to {
  transform: translateY(-150px);
  opacity: 0;
}
</style>
