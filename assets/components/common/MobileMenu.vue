<template>
  <nav class="fixed top-0 z-10 w-full border-b border-base-content/20 bg-base p-2" data-testid="navigation">
    <div class="flex items-center">
      <router-link :to="{ name: 'index' }">
        <svg class="h-14 w-28 fill-secondary">
          <use href="#logo"></use>
        </svg>
      </router-link>

      <div class="ml-auto flex items-center gap-2">
        <a class="btn btn-circle flex" @click="$emit('search')" :title="$t('tooltip.search')">
          <mdi:magnify class="h-5 w-5" />
        </a>
        <label class="btn btn-circle swap swap-rotate" data-testid="hamburger">
          <input type="checkbox" v-model="show" />
          <mdi:close class="swap-on" />
          <mdi:hamburger-menu class="swap-off" />
        </label>
      </div>
    </div>

    <transition name="fade">
      <div v-show="show">
        <div class="mt-4 flex items-center justify-center gap-2">
          <dropdown-menu
            v-model="sessionHost"
            :options="hosts"
            defaultLabel="Hosts"
            class="btn-sm"
            v-if="config.hosts.length > 1"
          />
          <router-link :to="{ name: 'settings' }" class="btn btn-outline btn-sm">
            <mdi:cog /> {{ $t("button.settings") }}
          </router-link>
          <a class="btn btn-outline btn-sm" :href="`${base}/logout`" :title="$t('button.logout')" v-if="secured">
            <mdi:logout /> {{ $t("button.logout") }}
          </a>
        </div>

        <ul class="menu">
          <li class="menu-title">{{ $t("label.containers") }}</li>
          <li v-for="item in sortedContainers" :key="item.id">
            <router-link
              :to="{ name: 'container-id', params: { id: item.id } }"
              active-class="active-primary"
              class="truncate"
              :title="item.name"
            >
              {{ item.name }}
            </router-link>
          </li>
        </ul>
      </div>
    </transition>
  </nav>
</template>

<script lang="ts" setup>
const { base, secured } = config;
import { sessionHost } from "@/composable/storage";
const store = useContainerStore();
const route = useRoute();
const { visibleContainers } = storeToRefs(store);

const show = ref(false);

watch(route, () => {
  show.value = false;
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

const hosts = computed(() => config.hosts.map(({ id, name }) => ({ value: id, label: name })));
</script>
<style scoped lang="postcss">
.fade-enter-active,
.fade-leave-active {
  @apply transition-opacity;
}

.fade-enter-active .menu,
.fade-leave-active .menu {
  @apply transition-transform;
}

.fade-enter-from,
.fade-leave-to {
  @apply opacity-0;
}

.fade-enter-from .menu,
.fade-leave-to .menu {
  @apply -translate-y-2;
}
</style>
