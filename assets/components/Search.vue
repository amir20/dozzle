<template lang="html">
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
        <span class="icon is-small is-left">
          <ion-icon name="search"></ion-icon>
        </span>
      </p>
    </div>
    <div class="column is-1 has-text-centered">
      <button class="delete is-medium" @click="resetSearch()"></button>
    </div>
  </div>
</template>

<script>
import { mapActions, mapState } from "vuex";
import hotkeys from "hotkeys-js";

export default {
  props: [],
  name: "Search",
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
  background: rgba(50, 50, 50, 0.9);
  top: 0;
  right: 0;
  border-radius: 0 0 0 5px;
  z-index: 10;
}
.delete {
  margin-left: 1em;
}
</style>
