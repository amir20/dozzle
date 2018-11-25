<template lang="html">
  <div class="columns is-marginless">
    <aside class="column menu is-2">
      <a
        role="button"
        class="navbar-burger burger is-white is-hidden-tablet is-pulled-right"
        @click="showNav = !showNav;"
        :class="{ 'is-active': showNav }"
      >
        <span></span> <span></span> <span></span>
      </a>
      <h1 class="title has-text-warning is-marginless">Dozzle</h1>
      <p class="menu-label is-hidden-mobile" :class="{ 'is-active': showNav }">Containers</p>
      <ul class="menu-list is-hidden-mobile" :class="{ 'is-active': showNav }">
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
      containers: [],
      showNav: false
    };
  },
  async created() {
    this.containers = await (await fetch(`${BASE_PATH}/api/containers.json`)).json();
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

  @media screen and (max-width: 768px) {
    & {
      position: sticky;
      top: 0;
      background: #222;
    }

    .tooltip::after,
    .tooltip::before {
      display: none !important;
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
