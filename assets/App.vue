<template lang="html">
  <div class="columns is-marginless">
    <aside class="column menu is-3-tablet is-2-widescreen">
      <a
        role="button"
        class="navbar-burger burger is-white is-hidden-tablet is-pulled-right"
        @click="showNav = !showNav"
        :class="{ 'is-active': showNav }"
      >
        <span></span> <span></span> <span></span>
      </a>
      <h1 class="title has-text-warning is-marginless">Dozzle</h1>
      <p class="menu-label is-hidden-mobile" :class="{ 'is-active': showNav }">Containers</p>
      <ul class="menu-list is-hidden-mobile" :class="{ 'is-active': showNav }">
        <li v-for="item in containers">
          <router-link :to="{ name: 'container', params: { id: item.id, name: item.name } }" active-class="is-active">
            <div class="hide-overflow">{{ item.name }}</div>
          </router-link>
        </li>
      </ul>
    </aside>
    <div class="column is-offset-3-tablet is-offset-2-widescreen is-9-tablet is-10-widescreen">
      <router-view></router-view>
    </div>
  </div>
</template>

<script>
let es;
export default {
  name: "App",
  data() {
    return {
      title: "",
      containers: [],
      showNav: false
    };
  },
  metaInfo() {
    return {
      title: this.title,
      titleTemplate: "%s - Dozzle"
    };
  },
  async created() {
    await this.fetchContainerList();
    es = new EventSource(`${BASE_PATH}/api/events/stream`);
    es.addEventListener("containers-changed", e => setTimeout(this.fetchContainerList, 1000), false);
  },
  beforeDestroy() {
    if (es) {
      es.close();
      es = null;
    }
  },
  methods: {
    async fetchContainerList() {
      this.containers = await (await fetch(`${BASE_PATH}/api/containers.json`)).json();
      this.title = `${this.containers.length} containers`;
    }
  },
  watch: {
    $route(to, from) {
      this.showNav = false;
    }
  }
};
</script>

<style scoped lang="scss">
.is-hidden-mobile.is-active {
  display: block !important;
}

.navbar-burger {
  height: 2.35rem;
}

aside {
  position: fixed;
  z-index: 2;
  padding: 1em;

  @media screen and (min-width: 769px) {
    & {
      height: 100vh;
      overflow: auto;
    }
  }

  @media screen and (max-width: 768px) {
    & {
      position: sticky;
      top: 0;
      left: 0;
      right: 0;
      background: #222;
    }

    .menu-label {
      margin-top: 1em;
    }
  }
}

.hide-overflow {
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow: hidden;
}

.burger.is-white {
  color: #fff;
}
</style>
