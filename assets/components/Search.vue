<template>
  <div class="search columns is-gapless is-vcentered" v-show="showSearch" v-if="settings.search">
    <div class="column">
      <p class="control has-icons-left">
        <input
          class="input"
          type="text"
          placeholder="Find / RegEx"
          ref="filter"
          v-model="filter"
          @keyup.esc="resetSearch()"
        />
        <span class="icon is-left">
          <search-icon />
        </span>
      </p>
    </div>
    <div class="column is-1 has-text-centered">
      <button class="delete is-medium" @click="resetSearch()"></button>
    </div>
  </div>
</template>

<script lang="ts">
import { mapActions, mapState } from "vuex";
import hotkeys from "hotkeys-js";
import SearchIcon from "~icons/mdi-light/magnify";

export default {
  props: [],
  name: "Search",
  components: {
    SearchIcon,
  },
  data() {
    return {
      showSearch: false,
    };
  },
  mounted() {
    hotkeys("command+f, ctrl+f", (event, handler) => {
      this.showSearch = true;
      this.$nextTick(() => this.$refs.filter.focus() || this.$refs.filter.select());
      event.preventDefault();
    });
    hotkeys("esc", (event, handler) => {
      this.resetSearch();
    });
  },
  beforeUnmount() {
    this.updateSearchFilter("");
    hotkeys.unbind("command+f, ctrl+f, esc");
  },
  methods: {
    ...mapActions({
      updateSearchFilter: "SET_SEARCH",
    }),
    resetSearch() {
      this.showSearch = false;
      this.filter = "";
    },
  },
  computed: {
    ...mapState(["searchFilter", "settings"]),
    filter: {
      get() {
        return this.searchFilter;
      },
      set(value) {
        this.updateSearchFilter(value);
      },
    },
  },
};
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
