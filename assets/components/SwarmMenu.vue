<template>
  <ul class="menu p-0 text-[0.95rem]">
    <li v-for="{ name, services } in stacks" :key="name">
      <details open>
        <summary class="font-light text-base-content/80">
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
            <router-link
              :to="{ name: '/service/[name]', params: { name: service.name } }"
              active-class="active-primary"
            >
              <ph:stack-simple />
              <div class="truncate">
                {{ service.name }}
              </div>
            </router-link>
          </li>
        </ul>
      </details>
    </li>

    <li v-if="serivcesWithoutStacks.length > 0">
      <details open>
        <summary class="font-light text-base-content/80">
          <ph:circles-four />
          {{ $t("label.services") }}
        </summary>
        <ul>
          <li v-for="service in serivcesWithoutStacks" :key="service.name">
            <router-link
              :to="{ name: '/service/[name]', params: { name: service.name } }"
              active-class="active-primary"
            >
              <ph:stack-simple />
              <div class="truncate">
                {{ service.name }}
              </div>
            </router-link>
          </li>
        </ul>
      </details>
    </li>

    <li v-if="customGroups.length > 0">
      <details open>
        <summary class="font-light text-base-content/80">
          <ph:bounding-box-fill />
          {{ $t("label.custom-groups") }}
        </summary>
        <ul>
          <li v-for="group in customGroups" :key="group.name">
            <router-link :to="{ name: '/group/[name]', params: { name: group.name } }" active-class="active-primary">
              <ph:stack-simple />
              <div class="truncate">
                {{ group.name }}
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

const { stacks, services, customGroups } = storeToRefs(store);

const serivcesWithoutStacks = computed(() => services.value.filter((service) => !service.stack));
</script>
<style scoped lang="postcss">
.menu {
  @apply text-[0.95rem];
}
</style>
