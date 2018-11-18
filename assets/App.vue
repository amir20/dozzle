<template lang="html">
  <div class="columns">
    <aside class="column menu is-2 section">
      <h1 class="title has-text-warning">Dozzle</h1>
      <p class="menu-label">Containers</p>
      <ul class="menu-list">
        <li v-for="item in containers">
          <router-link :to="{ name: 'container', params: { id: item.Id } }" active-class="is-active">
            {{ item.Names[0] }}
          </router-link>
        </li>
      </ul>
    </aside>
    <div class="column is-offset-2"><router-view></router-view></div>
  </div>
</template>

<script>
export default {
  name: "App",
  data() {
    return {
      containers: []
    };
  },
  async created() {
    this.containers = await (await fetch(`/api/containers.json`)).json();
  }
};
</script>

<style>
.section.is-fullwidth {
  padding: 0 !important;
}

aside {
  position: fixed;
}
</style>
