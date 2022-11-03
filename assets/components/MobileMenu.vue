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
      <div class="column is-narrow has-text-right is-2 push-right">
        <button class="button is-rounded" @click="$emit('search')" :title="$t('tooltip.search')">
          <span class="icon">
            <mdi-light-magnify />
          </span>
        </button>
      </div>
      <div class="column is-narrow has-text-right is-2 push-right">
        <router-link :to="{ name: 'settings' }" active-class="is-active" class="button is-rounded">
          <span class="icon">
            <mdi-light-cog />
          </span>
        </router-link>
      </div>
      <div class="column is-narrow has-text-right is-2 push-right" v-if="secured">
        <a :href="`${base}/logout`" active-class="is-active" class="button is-rounded">
          <span class="icon">
            <mdi-logout />
          </span>
        </a>
      </div>
      <div class="column ml-4 is-family-monospace is-ellipsis" v-if="$route.name == 'container'">
        {{ allContainersById[route.params.id].name }}
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

    <p class="menu-label is-hidden-mobile" :class="{ 'is-active': showNav }">{{ $t("label.containers") }}</p>
    <ul class="menu-list is-hidden-mobile" :class="{ 'is-active': showNav }">
      <li v-for="item in visibleContainers" :key="item.id">
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
const { base, secured }  = config;
const store = useContainerStore();
const route = useRoute();
const { visibleContainers, allContainersById } = storeToRefs(store);

let showNav = $ref(false);

watch(route, () => {
  showNav = false;
});
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
