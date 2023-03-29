<template>
  <div class="search columns is-gapless is-vcentered" v-show="showSearch" v-if="search">
    <div class="column">
      <p class="control has-icons-left">
        <input
          class="input"
          type="text"
          placeholder="Find / RegEx"
          ref="input"
          v-model="searchFilter"
          @keyup.esc="resetSearch()"
        />
        <span class="icon is-left">
          <mdi:light-magnify />
        </span>
      </p>
    </div>
    <div class="column is-1 has-text-centered">
      <button class="delete is-medium" @click="resetSearch()"></button>
    </div>
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

<style lang="scss" scoped>
.search {
  width: 350px;
  position: fixed;
  padding: 10px;
  background: var(--scheme-main-ter);
  top: 0;
  right: 0;
  border-radius: 0 0 0 5px;
  z-index: 10;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24);

  button.delete {
    margin-left: 1em;
    background-color: var(--scheme-main-ter);
    opacity: 0.6;

    &:after,
    &:before {
      background-color: var(--text-color);
    }

    &:hover {
      opacity: 1;
    }
  }

  .icon {
    padding: 10px 3px;
  }

  .input {
    color: var(--body-color);

    &::placeholder {
      color: var(--border-color);
    }
  }
}
</style>
