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
          <p class="heading">Containers</p>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <p class="title">{{ visibleContainers.length }}</p>
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

    <section class="columns is-centered section">
      <div class="column is-4">
        <div class="panel">
          <p class="panel-heading">
            Recent Containers
          </p>
          <router-link
            :to="{ name: 'container', params: { id: item.id, name: item.name } }"
            v-for="item in topMostRecent"
            :key="item.id"
            class="panel-block"
          >
            {{ item.name }}
          </router-link>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
export default {
  name: "Index",

  computed: {
    ...mapGetters(["visibleContainers"]),
    ...mapState(["containers"]),
    topMostRecent() {
      return this.visibleContainers.sort((a, b) => b.created - a.created).slice(0, 10);
    },
  },
};
</script>
<style lang="scss" scoped>
.panel {
  border: 1px solid var(--border-color);
  .panel-block {
    border-color: var(--border-color);
  }
}
</style>
