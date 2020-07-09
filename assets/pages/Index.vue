<template>
  <div>
    <section class="hero is-small mt-4">
      <div class="hero-body">
        <div class="container">
          <h1 class="title">
            Hello, friend.
          </h1>
          <h2 class="subtitle">
            I hope you are having a great day!
          </h2>
        </div>
      </div>
    </section>
    <section class="level section">
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
          <p class="title">1.24.1</p>
          <p class="heading">Dozzle Version</p>
        </div>
      </div>
    </section>

    <section class="columns is-centered">
      <div class="column is-4">
        <div class="panel">
          <p class="panel-heading">
            Containers
          </p>
          <div class="panel-block">
            <p class="control has-icons-left">
              <input class="input" type="text" placeholder="Search" />
              <span class="icon is-left">
                <icon name="search"></icon>
              </span>
            </p>
          </div>
          <p class="panel-tabs">
            <a :class="{ 'is-active': sort == 'all' }" @click="sort = 'all'">All</a>
            <a :class="{ 'is-active': sort == 'running' }" @click="sort = 'running'">Running</a>
            <a :class="{ 'is-active': sort == 'recent' }" @click="sort = 'recent'">Recent</a>
          </p>
          <router-link
            :to="{ name: 'container', params: { id: item.id, name: item.name } }"
            v-for="item in sortedContainers"
            :key="item.id"
            class="panel-block"
          >
            <span class="name">{{ item.name }}</span>
            <div class="subtitle is-7 status">
              {{ item.status }}
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
import { debug } from "webpack";
export default {
  name: "Index",
  components: { Icon },
  data() {
    return {
      search: null,
      sort: "all",
    };
  },
  computed: {
    ...mapState(["containers"]),
    mostRecentContainers() {
      return this.containers.sort((a, b) => b.created - a.created).slice(0, 5);
    },
    runningContainers() {
      return this.containers.filter((c) => c.state === "running").slice(0, 5);
    },
    allContainers() {
      return this.containers;
    },
    sortedContainers() {
      return [];
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
