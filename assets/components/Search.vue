<template lang="html">
  <div class="search columns is-gapless is-vcentered" v-show="showSearch">
    <div class="column">
      <p class="control has-icons-left">
        <input class="input" type="text" placeholder="Filter" ref="filter" v-model="filter" />
        <span class="icon is-small is-left"><i class="fas fa-search"></i></span>
      </p>
    </div>
    <div class="column is-1 has-text-centered">
      <button class="delete is-medium" @click="resetSearch()"></button>
    </div>
  </div>
</template>

<script>
import { mapActions, mapState } from "vuex";
export default {
  props: [],
  name: "Search",
  data() {
    return {
      showSearch: false
    };
  },
  mounted() {
    window.addEventListener("keydown", this.onKeyDown);
  },
  destroyed() {
    window.removeEventListener("keydown", this.onKeyDown);
  },
  methods: {
    ...mapActions({
      updateSearchFilter: "SET_SEARCH"
    }),
    onKeyDown(e) {
      if ((e.metaKey || e.ctrlKey) && e.key === "f") {
        this.showSearch = true;
        this.$nextTick(() => this.$refs.filter.focus());
        e.preventDefault();
      } else if (e.key === "Escape") {
        this.resetSearch();
      }
    },
    resetSearch() {
      this.showSearch = false;
      this.filter = "";
    }
  },
  computed: {
    ...mapState(["searchFilter"]),
    filter: {
      get() {
        return this.searchFilter;
      },
      set(value) {
        this.updateSearchFilter(value);
      }
    }
  }
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
