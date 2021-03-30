<template>
  <aside>
    <div class="columns is-marginless">
      <div class="column is-paddingless">
        <router-link :to="{ name: 'default' }">
          <svg class="logo">
            <use href="#logo"></use>
          </svg>
        </router-link>
      </div>
      <div class="column is-narrow has-text-right px-1">
        <button
          class="button is-small is-rounded is-settings-control"
          @click="$emit('search')"
          title="Search containers (⌘ + k, ⌃k)"
        >
          <span class="icon">
            <icon name="search"></icon>
          </span>
        </button>
      </div>
      <div class="column is-narrow has-text-right px-0">
        <router-link
          :to="{ name: 'settings' }"
          active-class="is-active"
          class="button is-small is-rounded is-settings-control"
        >
          <span class="icon">
            <icon name="cog"></icon>
          </span>
        </router-link>
      </div>
    </div>
    <p class="menu-label is-hidden-mobile">Containers</p>
    <ul class="menu-list is-hidden-mobile">
      <li v-for="item in visibleContainers" :key="item.id" :class="item.state">
        <router-link
          :to="{ name: 'container', params: { id: item.id, name: item.name } }"
          active-class="is-active"
          :title="item.name"
        >
          <div class="container is-flex is-align-items-center">
            <div class="is-flex-grow-1 is-ellipsis">
              {{ item.name }}
            </div>
            <div class="is-flex-shrink-1 column-icon">
              <span
                class="icon is-small"
                @click.stop.prevent="appendActiveContainer(item)"
                v-show="!activeContainersById[item.id]"
                title="Pin as column"
              >
                <icon name="column"></icon>
              </span>
            </div>
          </div>
        </router-link>
      </li>
    </ul>
  </aside>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";

import Icon from "./Icon";

export default {
  props: [],
  name: "SideMenu",
  components: {
    Icon,
  },
  data() {
    return {};
  },
  computed: {
    ...mapGetters(["visibleContainers", "activeContainers"]),
    activeContainersById() {
      return this.activeContainers.reduce((map, obj) => {
        map[obj.id] = obj;
        return map;
      }, {});
    },
  },
  methods: {
    ...mapActions({
      appendActiveContainer: "APPEND_ACTIVE_CONTAINER",
    }),
  },
};
</script>
<style scoped lang="scss">
aside {
  padding: 1em;
  height: 100vh;
  overflow: auto;
  position: fixed;
  width: inherit;

  .is-hidden-mobile.is-active {
    display: block !important;
  }
}

li.exited a {
  color: #777;
}

.logo {
  width: 122px;
  height: 54px;
  fill: var(--logo-color);
}

.menu-list li {
  .column-icon {
    visibility: hidden;
  }

  &:hover .column-icon {
    visibility: visible;
    &:hover {
      color: var(--secondary-color);
    }
  }
}
</style>
