<template>
  <div class="panel">
    <b-autocomplete
      ref="autocomplete"
      v-model="query"
      placeholder="Search containers using ⌘ + k, ⌃k"
      field="name"
      open-on-focus
      keep-first
      expanded
      :data="results"
      @select="selected"
    >
      <template slot-scope="props">
        <div class="media">
          <div class="media-left">
            <span class="icon is-small" :class="props.option.state">
              <container-icon />
            </span>
          </div>
          <div class="media-content">
            {{ props.option.name }}
          </div>
          <div class="media-right">
            <span class="icon is-small column-icon" @click.stop.prevent="addColumn(props.option)" title="Pin as column">
              <columns-icon />
            </span>
          </div>
        </div>
      </template>
    </b-autocomplete>
  </div>
</template>

<script>
import { mapState, mapActions } from "vuex";
import fuzzysort from "fuzzysort";

import PastTime from "./PastTime";
import ContainerIcon from "~icons/octicon/container-24";
import ColumnsIcon from "~icons/cil/columns";

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
    PastTime,
    ContainerIcon,
    ColumnsIcon,
  },
  mounted() {
    this.$nextTick(() => this.$refs.autocomplete.focus());
  },
  watch: {},
  methods: {
    ...mapActions({
      appendActiveContainer: "APPEND_ACTIVE_CONTAINER",
    }),
    selected(item) {
      this.$router.push({ name: "container", params: { id: item.id, name: item.name } });
      this.$emit("close");
    },
    addColumn(container) {
      this.appendActiveContainer(container);
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
        state: c.state,
        preparedName: fuzzysort.prepare(c.name),
      }));
    },
    results() {
      const options = {
        limit: this.maxResults,
        key: "preparedName",
      };
      if (this.query) {
        const results = fuzzysort.go(this.query, this.preparedContainers, options);
        results.forEach((result) => {
          if (result.obj.state === "running") {
            result.score += 1;
          }
        });
        return results.sort((a, b) => b.score - a.score).map((i) => i.obj);
      } else {
        return [...this.containers].sort((a, b) => b.created - a.created);
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.panel {
  min-height: 400px;
}

.running {
  color: var(--primary-color);
}

.exited {
  color: var(--scheme-main-ter);
}

.column-icon {
  &:hover {
    color: var(--secondary-color);
  }
}

::v-deep a.dropdown-item {
  padding-right: 1em;
  .media-right {
    visibility: hidden;
  }
  &:hover .media-right {
    visibility: visible;
  }
}

.icon {
  vertical-align: middle;
}
</style>
