<template>
  <div class="panel">
    <b-autocomplete
      ref="autocomplete"
      v-model="query"
      placeholder="Search containers"
      field="name"
      open-on-focus
      keep-first
      expanded
      :data="results"
      @select="selected"
    >
    </b-autocomplete>
  </div>
</template>

<script>
import { mapState } from "vuex";
import fuzzysort from "fuzzysort";

import PastTime from "./PastTime";
import Icon from "./Icon";

export default {
  props: {
    maxResults: {
      default: 20,
      type: Number,
    },
  },
  data() {
    return {
      query: "",
    };
  },
  name: "FuzzySearchModal",
  components: {
    Icon,
    PastTime,
  },
  mounted() {
    this.$nextTick(() => this.$refs.autocomplete.focus());
  },
  watch: {},
  methods: {
    selected(item) {
      this.$router.push({ name: "container", params: { id: item.id, name: item.name } });
      this.$emit("close");
    },
  },
  computed: {
    ...mapState(["containers"]),
    preparedContainers() {
      return this.containers.map((c) => ({
        name: c.name,
        id: c.id,
        created: c.created,
        preparedName: fuzzysort.prepare(c.name),
      }));
    },
    results() {
      const options = {
        limit: this.maxResults,
        key: "preparedName",
      };
      if (this.query) return fuzzysort.go(this.query, this.preparedContainers, options).map((i) => i.obj);
      else {
        return [...this.containers].sort((a, b) => b.created - a.created);
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.panel {
  height: 400px;
}
</style>
