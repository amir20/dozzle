<template>
  <aside>
    <div class="columns is-marginless">
      <div class="column is-paddingless">
        <router-link :to="{ name: 'index' }">
          <svg class="logo">
            <use href="#logo"></use>
          </svg>
        </router-link>
      </div>
      <div class="column is-narrow has-text-right px-1">
        <button class="button is-rounded" @click="$emit('search')" title="Search containers (⌘ + k, ⌃k)">
          <span class="icon">
            <mdi-light-magnify />
          </span>
        </button>
      </div>
      <div class="column is-narrow has-text-right px-0">
        <router-link :to="{ name: 'settings' }" active-class="is-active" class="button is-rounded">
          <span class="icon">
            <mdi-light-cog />
          </span>
        </router-link>
      </div>
    </div>
    <p class="menu-label is-hidden-mobile">Containers</p>
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
                title="Pin as column"
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
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { useContainerStore } from "@/stores/container";
import type { Container } from "@/types/Container";

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
