<template lang="html">
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
          <span @click.stop.prevent="appendActiveContainer(item)"><i class="fas fa-thumbtack"></i></span>
        </router-link>
      </li>
    </ul>
  </aside>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";

export default {
  props: [],
  name: "SideMenu",
  data() {
    return {
      showNav: false
    };
  },
  methods: {
    colorize: function(value) {
      return ansiConvertor
        .toHtml(value)
        .replace("&lt;mark&gt;", "<mark>")
        .replace("&lt;/mark&gt;", "</mark>");
    }
  },
  computed: {
    ...mapState(["containers", "activeContainers"])
  },
  methods: {
    ...mapActions({
      appendActiveContainer: "APPEND_ACTIVE_CONTAINER"
    })
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
