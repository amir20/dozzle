<template>
  <aside>
    <div class="columns is-marginless">
      <div class="column is-paddingless">
        <h1>
          <router-link :to="{ name: 'index' }">
            <svg class="logo">
              <use href="#logo"></use>
            </svg>
          </router-link>

          <small class="subtitle is-6 is-block mb-4" v-if="hostname">
            {{ hostname }}
          </small>
        </h1>
        <div v-if="config.hosts.length > 1" class="mb-3">
          <o-dropdown v-model="sessionHost" aria-role="list">
            <template #trigger>
              <o-button variant="primary" type="button" size="small">
                <span>{{ sessionHost }}</span>
                <span class="icon">
                  <carbon-caret-down />
                </span>
              </o-button>
            </template>

            <o-dropdown-item :value="value" aria-role="listitem" v-for="value in config.hosts" :key="value">
              <span>{{ value }}</span>
            </o-dropdown-item>
          </o-dropdown>
        </div>
      </div>
    </div>
    <div class="columns is-marginless">
      <div class="column is-narrow py-0 pl-0 pr-1">
        <button class="button is-rounded is-small" @click="$emit('search')" :title="$t('tooltip.search')">
          <span class="icon">
            <mdi-light-magnify />
          </span>
        </button>
      </div>
      <div class="column is-narrow py-0" :class="secured ? 'pl-0 pr-1' : 'px-0'">
        <router-link :to="{ name: 'settings' }" active-class="is-active" class="button is-rounded is-small">
          <span class="icon">
            <mdi-light-cog />
          </span>
        </router-link>
      </div>
      <div class="column is-narrow py-0 px-0" v-if="secured">
        <a class="button is-rounded is-small" :href="`${base}/logout`" :title="$t('button.logout')">
          <span class="icon">
            <mdi-light-logout />
          </span>
        </a>
      </div>
    </div>
    <p class="menu-label is-hidden-mobile">{{ $t("label.containers") }}</p>
    <ul class="menu-list is-hidden-mobile" v-if="ready">
      <li v-for="item in visibleContainers" :key="item.id" :class="item.state">
        <router-link
          :to="{ name: 'container-id', params: { id: item.id } }"
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
                @click.stop.prevent="store.appendActiveContainer(item)"
                v-show="!activeContainersById[item.id]"
                :title="$t('tooltip.pin-column')"
              >
                <cil-columns />
              </span>
            </div>
          </div>
        </router-link>
      </li>
    </ul>
    <ul class="menu-list is-hidden-mobile loading" v-else>
      <li v-for="index in 7" class="my-4"><o-skeleton animated size="large" :key="index"></o-skeleton></li>
    </ul>
  </aside>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { sessionHost } from "@/composables/storage";

const { base, secured, hostname } = config;
const store = useContainerStore();

const { activeContainers, visibleContainers, ready } = storeToRefs(store);

const activeContainersById = computed(() =>
  activeContainers.value.reduce((acc, item) => {
    acc[item.id] = item;
    return acc;
  }, {} as Record<string, Container>)
);
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

.loading {
  opacity: 0.5;
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

    & > span {
      vertical-align: middle;
    }
  }

  &:hover .column-icon {
    visibility: visible;

    &:hover {
      color: var(--secondary-color);
    }
  }
}
</style>
