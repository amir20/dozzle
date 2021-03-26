<template>
  <div>
    <section class="hero is-small mt-4">
      <div class="hero-body">
        <div class="container">
          <h1 class="title">Hello, there!</h1>
        </div>
      </div>
    </section>
    <section class="level section is-mobile">
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ containers.length }}</p>
          <p class="heading">Total Containers</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ runningContainers.length }}</p>
          <p class="heading">Running</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ version }}</p>
          <p class="heading">Dozzle Version</p>
        </div>
      </div>
    </section>

    <section class="columns is-centered section">
      <div class="column is-4">
        <div class="panel">
          <p class="panel-heading">Containers</p>
          <div class="panel-block">
            <p class="control has-icons-left">
              <input
                class="input"
                type="text"
                placeholder="Search Containers"
                v-model="search"
                @keyup.esc="search = null"
                @keyup.enter="onEnter()"
              />
              <span class="icon is-left">
                <icon name="search"></icon>
              </span>
            </p>
          </div>
          <p class="panel-tabs" v-if="!search">
            <a :class="{ 'is-active': sort === 'running' }" @click="sort = 'running'">Running</a>
            <a :class="{ 'is-active': sort === 'all' }" @click="sort = 'all'">All</a>
          </p>
          <router-link
            :to="{ name: 'container', params: { id: item.id, name: item.name } }"
            v-for="item in results.slice(0, 10)"
            :key="item.id"
            class="panel-block"
          >
            <span class="name">{{ item.name }}</span>

            <div class="subtitle is-7 status">
              <past-time :date="new Date(item.created * 1000)"></past-time>
            </div>
          </router-link>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
import Icon from "../components/Icon";
import PastTime from "../components/PastTime";
import config from "../store/config";
import fuzzysort from "fuzzysort";

export default {
  name: "Index",
  components: { Icon, PastTime },
  data() {
    return {
      version: config.version,
      search: null,
      sort: "running",
    };
  },
  methods: {
    onEnter() {
      if (this.results.length == 1) {
        const [item] = this.results;
        this.$router.push({ name: "container", params: { id: item.id, name: item.name } });
      }
    },
  },
  computed: {
    ...mapState(["containers"]),
    mostRecentContainers() {
      return [...this.containers].sort((a, b) => b.created - a.created);
    },
    runningContainers() {
      return this.mostRecentContainers.filter((c) => c.state === "running");
    },
    allContainers() {
      return this.containers;
    },
    results() {
      if (this.search) {
        const term = this.search.toLowerCase();
        return fuzzysort.go(term, this.allContainers, { key: "name" }).map((i) => i.obj);
      }
      switch (this.sort) {
        case "all":
          return this.mostRecentContainers;
        case "running":
          return this.runningContainers;

        default:
          throw `Invalid sort order: ${this.sort}`;
      }
    },
  },
};
</script>
<style lang="scss" scoped>
.panel {
  border: 1px solid var(--border-color);
  .panel-block,
  .panel-tabs {
    border-color: var(--border-color);
    .is-active {
      border-color: var(--border-hover-color);
    }
    .name {
      text-overflow: ellipsis;
      white-space: nowrap;
      overflow: hidden;
    }
    .status {
      margin-left: auto;
      white-space: nowrap;
    }
  }
}

.icon {
  padding: 10px 3px;
}
</style>
