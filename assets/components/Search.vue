<template>
  <div
    class="fixed -right-px -top-px z-10 flex w-96 items-center gap-4 rounded-bl border border-secondary/20 bg-base-darker p-4 shadow"
    v-show="showSearch"
    v-if="search"
  >
    <div class="input input-primary flex h-auto items-center">
      <mdi:magnify />
      <input
        class="input flex-1"
        type="text"
        placeholder="Find / RegEx"
        ref="input"
        v-model="searchFilter"
        @keyup.esc="resetSearch()"
      />
    </div>

    <a class="btn btn-circle btn-xs" @click="resetSearch()"> <mdi:close /></a>
  </div>
</template>

<script lang="ts" setup>
const input = ref<HTMLInputElement>();
const { searchFilter, showSearch, resetSearch } = useSearchFilter();

onKeyStroke("f", (e) => {
  if (e.ctrlKey || e.metaKey) {
    showSearch.value = true;
    nextTick(() => input.value?.focus() || input.value?.select());
    e.preventDefault();
  }
});

onUnmounted(() => resetSearch());
</script>
