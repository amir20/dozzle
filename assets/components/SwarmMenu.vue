<template>
  <ul class="menu w-full p-0 text-[0.95rem]">
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
</script>
