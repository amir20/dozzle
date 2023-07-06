<template>
  <aside>
    <div class="columns is-marginless is-gapless is-mobile is-vcentered">
      <div class="column is-narrow">
        <router-link :to="{ name: 'index' }">
          <svg class="logo">
            <use href="#logo"></use>
          </svg>
        </router-link>
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

    <div class="menu-label level is-mobile is-hidden-mobile" :class="{ 'is-active': showNav }">
      <div v-if="config.hosts.length > 1">
        <o-dropdown v-model="sessionHost" aria-role="list">
          <template #trigger>
            <o-button variant="primary" type="button" size="small">
              <span>{{ sessionHost ? hosts[sessionHost].name : "" }}</span>
              <span class="icon">
                <carbon:caret-down />
              </span>
            </o-button>
          </template>

          <o-dropdown-item :value="value.host" aria-role="listitem" v-for="value in config.hosts" :key="value">
            <span>{{ value.name }}</span>
          </o-dropdown-item>
        </o-dropdown>
      </div>

      <div class="level-item has-text-centered">
        <div>
          <button class="button is-small is-rounded" @click="$emit('search')" :title="$t('tooltip.search')">
            <span class="icon">
              <mdi:light-magnify />
            </span>
          </button>
        </div>
      </div>
      <div class="level-item has-text-centered">
        <div>
          <router-link :to="{ name: 'settings' }" active-class="is-active" class="button is-small is-rounded">
            <span class="icon">
              <mdi:light-cog />
            </span>
          </router-link>
        </div>
      </div>
      <div class="level-item has-text-centered" v-if="secured">
        <div>
          <a class="button is-small is-rounded" :href="`${base}/logout`" :title="$t('button.logout')">
            <span class="icon">
              <mdi:light-logout />
            </span>
          </a>
        </div>
      </div>
    </div>

    <p class="menu-label is-hidden-mobile" :class="{ 'is-active': showNav }">{{ $t("label.containers") }}</p>
    <ul class="menu-list is-hidden-mobile" :class="{ 'is-active': showNav }">
      <li v-for="item in sortedContainers" :key="item.id">
        <router-link
          :to="{ name: 'container-id', params: { id: item.id } }"
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

<script lang="ts" setup>
const { base, secured } = config;
import { sessionHost } from "@/composables/storage";
const store = useContainerStore();
const route = useRoute();
const { visibleContainers } = storeToRefs(store);

let showNav = $ref(false);

watch(route, () => {
  showNav = false;
});

const sortedContainers = computed(() =>
  visibleContainers.value
    .filter((c) => c.host === sessionHost.value)
    .sort((a, b) => {
      if (a.state === "running" && b.state !== "running") {
        return -1;
      } else if (a.state !== "running" && b.state === "running") {
        return 1;
      } else {
        return a.name.localeCompare(b.name);
      }
    }),
);

const hosts = computed(() =>
  config.hosts.reduce(
    (acc, item) => {
      acc[item.host] = item;
      return acc;
    },
    {} as Record<string, { name: string; host: string }>,
  ),
);
</script>
<style scoped lang="scss">
aside {
  padding: 1em;
  position: fixed;
  left: 0;
  right: 0;
  background: var(--scheme-main-ter);
  z-index: 10;
  max-height: 100vh;
  overflow: auto;

  .level.is-hidden-mobile.is-active {
    display: flex !important;
  }

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
