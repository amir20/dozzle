<template lang="html">
  <aside>
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
        <router-link
          :to="{ name: 'container', params: { id: item.id, name: item.name } }"
          active-class="is-active"
          :title="item.name"
        >
          <div class="hide-overflow">
            {{ item.name }}
          </div>
        </router-link>
      </li>
    </ul>
  </aside>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";

export default {
  props: [],
  name: "MobileMenu",
  data() {
    return {
      showNav: false,
    };
  },

  computed: {
    ...mapState(["containers"]),
    ...mapGetters(["activeContainersById"]),
  },
  methods: {
    ...mapActions({}),
  },
  watch: {
    $route(to, from) {
      this.showNav = false;
    },
  },
};
</script>
<style scoped lang="scss">
aside {
  padding: 1em;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  background: #222;
  z-index: 2;
  max-height: 100vh;
  overflow: auto;

  .menu-label {
    margin-top: 1em;
  }
  .hide-overflow {
    text-overflow: ellipsis;
    white-space: nowrap;
    overflow: hidden;
  }

  .burger.is-white {
    color: #fff;
  }

  .is-hidden-mobile.is-active {
    display: block !important;
  }

  .navbar-burger {
    height: 2.35rem;
  }
}
</style>
