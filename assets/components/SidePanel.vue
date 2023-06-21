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
                  <carbon:caret-down />
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
            <mdi:light-magnify />
          </span>
        </button>
      </div>
      <div class="column is-narrow py-0" :class="secured ? 'pl-0 pr-1' : 'px-0'">
        <router-link
          :to="{ name: 'settings' }"
          active-class="is-active"
          class="button is-rounded is-small"
          :aria-label="$t('title.settings')"
        >
          <span class="icon">
            <mdi:light-cog />
          </span>
        </router-link>
      </div>
      <div class="column is-narrow py-0 px-0" v-if="secured">
        <a class="button is-rounded is-small" :href="`${base}/logout`" :title="$t('button.logout')">
          <span class="icon">
            <mdi:light-logout />
          </span>
        </a>
      </div>
    </div>
    <side-menu class="mt-4"></side-menu>
  </aside>
</template>

<script lang="ts" setup>
import { sessionHost } from "@/composables/storage";

const { base, secured, hostname } = config;
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

.logo {
  width: 122px;
  height: 54px;
  fill: var(--logo-color);
}
</style>
