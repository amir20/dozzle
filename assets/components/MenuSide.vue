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
      <div class="column is-narrow has-text-right x">
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
      <li
        v-for="item in visibleContainers"
        :key="item.id"
        :class="{
          [item.state]: true,
          'high-cpu': item.stat.cpu > settings.cpuThreshold,
          'high-mem': item.stat.memory > settings.memoryThreshold,
        }"
      >
        <router-link
          :to="{ name: 'container', params: { id: item.id, name: item.name } }"
          active-class="is-active"
          :title="item.name"
        >
          <div class="is-ellipsis">
            <span
              @click.stop.prevent="appendActiveContainer(item)"
              class="icon is-small will-append-container"
              :class="{ 'is-active': activeContainersById[item.id] }"
            >
              <icon name="pin"></icon>
            </span>
            {{ item.name }}
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
  name: "MenuSide",
  components: {
    Icon,
  },
  data() {
    return {};
  },
  computed: {
    ...mapState(["settings"]),
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
li {
  &.exited a {
    color: #777;
  }

  &.high-cpu a,
  &.high-mem a {
    color: var(--danger-color);
  }
}

.logo {
  width: 122px;
  height: 54px;
  fill: var(--logo-color);
}

.will-append-container.icon {
  transition: transform 0.2s ease-out;
  &.is-active {
    pointer-events: none;
    color: var(--primary-color);
  }
  .router-link-exact-active & {
    visibility: hidden;
  }
}
</style>
