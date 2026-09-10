<template>
  <NavHeader :title="$t('label.service', services.length)">
    <template #actions>
      <NavOverflow>
        <button type="button" class="nav-menu-item" @click="toggleAll()">
          <material-symbols-light:expand-all class="size-4 shrink-0 opacity-60" v-if="allCollapsed" />
          <material-symbols-light:collapse-all class="size-4 shrink-0 opacity-60" v-else />
          {{ allCollapsed ? $t("label.expand-all") : $t("label.collapse-all") }}
        </button>
      </NavOverflow>
    </template>
  </NavHeader>

  <ul class="space-y-1">
    <NavGroup
      v-for="{ name, services } in stacks"
      :key="name"
      :label="name"
      :count="services.length"
      :icon="Stack"
      :open="!collapsed.has(name)"
      @update:open="setCollapsed(name, $event)"
    >
      <template #actions>
        <NavMergeLink :to="{ name: '/stack/[name]', params: { name } }" />
      </template>
      <NavItem
        v-for="service in services"
        :key="service.name"
        :to="{ name: '/service/[name]', params: { name: service.name } }"
        :label="service.name"
        :title="service.name"
      >
        <template #icon><ph:stack-simple class="size-4 opacity-70" /></template>
      </NavItem>
    </NavGroup>

    <NavGroup
      v-if="servicesWithoutStacks.length > 0"
      :label="$t('label.services')"
      :count="servicesWithoutStacks.length"
      :icon="CirclesFour"
      :open="!collapsed.has(UNGROUPED)"
      @update:open="setCollapsed(UNGROUPED, $event)"
    >
      <NavItem
        v-for="service in servicesWithoutStacks"
        :key="service.name"
        :to="{ name: '/service/[name]', params: { name: service.name } }"
        :label="service.name"
        :title="service.name"
      >
        <template #icon><ph:stack-simple class="size-4 opacity-70" /></template>
      </NavItem>
    </NavGroup>
  </ul>
</template>

<script lang="ts" setup>
import Stack from "~icons/ph/stack";
import CirclesFour from "~icons/ph/circles-four";

const store = useSwarmStore();

const { stacks, services } = storeToRefs(store);

const servicesWithoutStacks = computed(() => services.value.filter((service) => !service.stack));

/** Stand-in key for the stackless bucket, which has no stack name of its own. */
const UNGROUPED = "__services__";

const collapsed = ref(new Set<string>());

const setCollapsed = (key: string, open: boolean) => {
  const next = new Set(collapsed.value);
  open ? next.delete(key) : next.add(key);
  collapsed.value = next;
};

const groupKeys = computed(() => [
  ...stacks.value.map(({ name }) => name),
  ...(servicesWithoutStacks.value.length > 0 ? [UNGROUPED] : []),
]);

const allCollapsed = computed(() => groupKeys.value.length > 0 && groupKeys.value.every((k) => collapsed.value.has(k)));

const toggleAll = () => {
  collapsed.value = allCollapsed.value ? new Set() : new Set(groupKeys.value);
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur();
  }
};
</script>
