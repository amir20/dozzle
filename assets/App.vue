<template lang="html">
  <div class="columns is-marginless">
    <aside class="column menu is-2 section">
      <h1 class="title has-text-warning">Dozzle</h1>
      <p class="menu-label">Containers</p>
      <ul class="menu-list">
        <li v-for="item in containers">
          <router-link
            :to="{ name: 'container', params: { id: item.Id } }"
            active-class="is-active"
            class="tooltip is-tooltip-right is-tooltip-info"
            :data-tooltip="item.Names[0]"
          >
            <div class="hide-overflow">{{ item.Names[0] }}</div>
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
    this.containers = await (await fetch(`${BASE_PATH}/api/containers.json`)).json();
  }
};
</script>

<style scoped>
aside {
  position: fixed;
  padding-right: 0;
}

.hide-overflow {
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow: hidden;
}
</style>
