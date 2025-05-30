<template>
  <div class="mb-2 flex items-center">
    <div class="flex-1">
      {{ $t("label.service", services.length) }}
    </div>
    <div class="flex-none">
      <div class="dropdown dropdown-end dropdown-hover">
        <label tabindex="0" class="btn btn-square btn-ghost btn-sm">
          <ph:dots-three-vertical-bold />
        </label>
        <ul
          tabindex="0"
          class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 w-52 border p-1 shadow-sm"
        >
          <li>
            <a class="text-sm capitalize" @click="collapseAll()">
              <material-symbols-light:collapse-all class="w-4" />
              {{ $t("label.collapse-all") }}
            </a>
          </li>
        </ul>
      </div>
    </div>
  </div>
  <ul class="menu w-full p-0 text-[0.95rem]" ref="menu">
    <li v-for="{ name, services } in stacks" :key="name">
      <details open>
        <summary class="text-base-content/80 font-light">
          <ph:stack />
          {{ name }}

          <router-link
            :to="{ name: '/stack/[name]', params: { name } }"
            class="btn btn-square btn-outline btn-primary btn-xs"
            active-class="btn-active"
            :title="$t('tooltip.merge-services')"
          >
            <ph:arrows-merge />
          </router-link>
        </summary>
        <ul>
          <li v-for="service in services" :key="service.name">
            <router-link :to="{ name: '/service/[name]', params: { name: service.name } }" active-class="menu-active">
              <ph:stack-simple />
              <div class="truncate">
                {{ service.name }}
              </div>
            </router-link>
          </li>
        </ul>
      </details>
    </li>

    <li v-if="servicesWithoutStacks.length > 0">
      <details open>
        <summary class="text-base-content/80 font-light">
          <ph:circles-four />
          {{ $t("label.services") }}
        </summary>
        <ul>
          <li v-for="service in servicesWithoutStacks" :key="service.name">
            <router-link :to="{ name: '/service/[name]', params: { name: service.name } }" active-class="menu-active">
              <ph:stack-simple />
              <div class="truncate">
                {{ service.name }}
              </div>
            </router-link>
          </li>
        </ul>
      </details>
    </li>
  </ul>
</template>

<script lang="ts" setup>
const store = useSwarmStore();

const { stacks, services } = storeToRefs(store);

const servicesWithoutStacks = computed(() => services.value.filter((service) => !service.stack));

const menu = useTemplateRef("menu");

const collapseAll = () => {
  const details = menu.value?.querySelectorAll("details");
  details?.forEach((detail) => (detail.open = false));
};
</script>
