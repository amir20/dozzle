<template>
  <NavHeader :title="$t('label.service', services.length)" filterable v-model:filter="query">
    <template #actions>
      <NavOverflow v-if="!filtering">
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
      v-for="{ name, services } in visibleStacks"
      :key="name"
      :label="name"
      :count="services.length"
      :icon="Stack"
      :open="isOpen(name, !collapsed.has(name))"
      @update:open="setOpen(name, $event, (open) => setCollapsed(name, open))"
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
      v-if="visibleUngrouped.length > 0"
      :label="$t('label.services')"
      :count="visibleUngrouped.length"
      :icon="CirclesFour"
      :open="isOpen(UNGROUPED, !collapsed.has(UNGROUPED))"
      @update:open="setOpen(UNGROUPED, $event, (open) => setCollapsed(UNGROUPED, open))"
    >
      <NavItem
        v-for="service in visibleUngrouped"
        :key="service.name"
        :to="{ name: '/service/[name]', params: { name: service.name } }"
        :label="service.name"
        :title="service.name"
      >
        <template #icon><ph:stack-simple class="size-4 opacity-70" /></template>
      </NavItem>
    </NavGroup>
  </ul>
  <p class="text-base-content/40 px-2 py-1 text-sm" v-if="filtering && nothingMatches">
    {{ $t("label.no-matches") }}
  </p>
</template>

<script lang="ts" setup>
import Stack from "~icons/ph/stack";
import CirclesFour from "~icons/ph/circles-four";

const store = useSwarmStore();

const { stacks, services } = storeToRefs(store);

const servicesWithoutStacks = computed(() => services.value.filter((service) => !service.stack));

const { query, active: filtering, matches, filterGroups, isOpen, setOpen } = useListFilter();

const visibleStacks = computed(() =>
  filterGroups(
    // Plain rows rather than Stack instances: building a Stack re-parents its services.
    stacks.value.map(({ name, services }) => ({ name, services })),
    (stack) => stack.name,
    (stack) => stack.services,
    (service) => service.name,
    (stack, services) => ({ ...stack, services }),
  ),
);

const visibleUngrouped = computed(() => servicesWithoutStacks.value.filter((s) => matches(s.name)));

const nothingMatches = computed(() => visibleStacks.value.length === 0 && visibleUngrouped.value.length === 0);

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
