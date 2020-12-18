<template>
  <aside>
    <div class="columns is-marginless is-gapless is-mobile is-vcentered">
      <div class="column is-narrow">
        <router-link :to="{ name: 'default' }">
          <svg class="logo">
            <use href="#logo"></use>
          </svg>
        </router-link>
      </div>
      <div class="column ml-4 is-family-monospace is-ellipsis" v-if="$route.name == 'container'">
        {{ allContainersById[$route.params.id].name }}
      </div>

      <div class="column is-narrow push-right">
        <a
          role="button"
          class="navbar-burger burger is-hidden-tablet is-pulled-right"
          @click="showNav = !showNav"
          :class="{ 'is-active': showNav }"
        >
          <span></span> <span></span> <span></span>
        </a>
      </div>
    </div>

    <p class="menu-label is-hidden-mobile" :class="{ 'is-active': showNav }">Containers</p>
    <ul class="menu-list is-hidden-mobile" :class="{ 'is-active': showNav }">
      <li v-for="item in visibleContainers" :key="item.id">
        <router-link
          :to="{ name: 'container', params: { id: item.id, name: item.name } }"
          active-class="is-active"
          :title="item.name"
        >
          <div class="is-ellipsis">
            {{ item.name }}
          </div>
        </router-link>
      </li>
    </ul>
  </aside>
</template>

<script>
import { mapGetters } from "vuex";

export default {
  props: [],
  name: "MenuMobile",
  data() {
    return {
      showNav: false,
    };
  },
  computed: {
    ...mapGetters(["visibleContainers", "allContainersById"]),
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
  background: var(--scheme-main-ter);
  z-index: 2;
  max-height: 100vh;
  overflow: auto;

  .menu-label {
    margin-top: 1em;
  }

  .title {
    text-shadow: 0px 1px 1px rgba(0, 0, 0, 0.2);
  }

  .burger {
    color: var(--body-color);
  }

  .is-hidden-mobile.is-active {
    display: block !important;
  }

  .navbar-burger {
    height: 2.35rem;
  }

  .logo {
    width: 82px;
    height: 36px;
    fill: var(--logo-color);
  }

  .column.push-right {
    margin-left: auto;
  }
}
</style>
