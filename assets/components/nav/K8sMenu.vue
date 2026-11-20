<template>
  <NavHeader
    :title="selectedNamespace === 'all' ? $t('label.all-namespaces') : $t('label.namespaces')"
    :back="selectedNamespace !== null ? $t('label.namespaces') : undefined"
    @back="setNamespace(null)"
    filterable
    v-model:filter="query"
  >
    <template #title v-if="selectedNamespace && selectedNamespace !== 'all'">
      <ph:circles-four class="text-base-content/50 size-4 shrink-0" />
      <span class="truncate text-[0.9375rem] font-medium">{{ selectedNamespace }}</span>
    </template>

    <template #actions>
      <NavMergeLink
        v-if="selectedNamespace && selectedNamespace !== 'all'"
        :to="{ name: '/namespace/[name]', params: { name: selectedNamespace } }"
      />
      <NavOverflow>
        <button type="button" class="nav-menu-item" @click="showAllContainers = !showAllContainers">
          <mdi:check class="size-4 shrink-0" v-if="showAllContainers" />
          <span class="size-4 shrink-0" v-else></span>
          {{ $t("label.show-all-containers") }}
        </button>
        <button type="button" class="nav-menu-item" @click="toggleAll()">
          <material-symbols-light:expand-all class="size-4 shrink-0 opacity-60" v-if="allCollapsed" />
          <material-symbols-light:collapse-all class="size-4 shrink-0 opacity-60" v-else />
          {{ allCollapsed ? $t("label.expand-all") : $t("label.collapse-all") }}
        </button>
      </NavOverflow>
    </template>
  </NavHeader>

  <SlideTransition :slide-right="selectedNamespace !== null">
    <template #left>
      <ul class="space-y-px">
        <NavItem :label="$t('label.all-namespaces')" @click="setNamespace('all')">
          <template #icon><ph:circles-four class="size-4" /></template>
        </NavItem>
        <NavItem
          v-for="ns in namespaces.filter((ns) => matches(ns.name))"
          :key="ns.name"
          :label="ns.name"
          :title="ns.name"
          @click="setNamespace(ns.name)"
        >
          <template #icon><ph:circles-four class="size-4" /></template>
        </NavItem>
      </ul>
    </template>

    <template #right>
      <ul class="space-y-1">
        <NavGroup
          v-for="{ name, owners } in visibleNamespaces"
          :key="name"
          :label="name"
          :count="owners.length"
          :icon="Stack"
          :open="isOpen(name, !collapsed.has(name))"
          @update:open="setOpen(name, $event, (open) => setCollapsed(name, open))"
        >
          <template #actions>
            <NavMergeLink :to="{ name: '/namespace/[name]', params: { name } }" />
          </template>
          <NavItem
            v-for="owner in owners"
            :key="owner.key"
            :to="{ name: '/owner/[name]', params: { name: owner.key } }"
            :label="`${owner.kind}/${owner.name}`"
            :title="`${owner.kind}/${owner.name}`"
          >
            <template #icon><ph:stack-simple class="size-4 opacity-70" /></template>
          </NavItem>
        </NavGroup>

        <NavGroup
          v-if="visibleUngrouped.length > 0"
          :label="$t('label.owners')"
          :count="visibleUngrouped.length"
          :icon="CirclesFour"
          :open="isOpen(UNGROUPED, !collapsed.has(UNGROUPED))"
          @update:open="setOpen(UNGROUPED, $event, (open) => setCollapsed(UNGROUPED, open))"
        >
          <NavItem
            v-for="owner in visibleUngrouped"
            :key="owner.key"
            :to="{ name: '/owner/[name]', params: { name: owner.key } }"
            :label="`${owner.kind}/${owner.name}`"
            :title="`${owner.kind}/${owner.name}`"
          >
            <template #icon><ph:stack-simple class="size-4 opacity-70" /></template>
          </NavItem>
        </NavGroup>
      </ul>
      <p class="text-base-content/40 px-2 py-1 text-sm" v-if="filtering && nothingMatches">
        {{ $t("label.no-matches") }}
      </p>
    </template>
  </SlideTransition>
</template>

<script lang="ts" setup>
import Stack from "~icons/ph/stack";
import CirclesFour from "~icons/ph/circles-four";
import { showAllContainers } from "@/stores/settings";

const store = useK8sStore();

const { namespaces, owners } = storeToRefs(store);

const selectedNamespace = ref<string | null>("all");

const setNamespace = (namespace: string | null) => (selectedNamespace.value = namespace);

const filteredNamespaces = computed(() => {
  if (selectedNamespace.value === null || selectedNamespace.value === "all") {
    return namespaces.value;
  }
  return namespaces.value.filter((ns) => ns.name === selectedNamespace.value);
});

const ownersWithoutNamespace = computed(() => {
  const filtered = owners.value.filter((owner) => !owner.namespace);
  if (selectedNamespace.value === null || selectedNamespace.value === "all") {
    return filtered;
  }
  return [];
});

const { query, active: filtering, matches, filterGroups, isOpen, setOpen } = useListFilter();

const ownerLabel = (owner: { kind: string; name: string }) => `${owner.kind}/${owner.name}`;

const visibleNamespaces = computed(() =>
  filterGroups(
    filteredNamespaces.value.map(({ name, owners }) => ({ name, owners })),
    (ns) => ns.name,
    (ns) => ns.owners,
    ownerLabel,
    (ns, owners) => ({ ...ns, owners }),
  ),
);

const visibleUngrouped = computed(() => ownersWithoutNamespace.value.filter((o) => matches(ownerLabel(o))));

const nothingMatches = computed(() => visibleNamespaces.value.length === 0 && visibleUngrouped.value.length === 0);

/** Stand-in key for the bucket of owners that belong to no namespace. */
const UNGROUPED = "__owners__";

const collapsed = ref(new Set<string>());

const setCollapsed = (key: string, open: boolean) => {
  const next = new Set(collapsed.value);
  open ? next.delete(key) : next.add(key);
  collapsed.value = next;
};

const groupKeys = computed(() => [
  ...filteredNamespaces.value.map(({ name }) => name),
  ...(ownersWithoutNamespace.value.length > 0 ? [UNGROUPED] : []),
]);

const allCollapsed = computed(() => groupKeys.value.length > 0 && groupKeys.value.every((k) => collapsed.value.has(k)));

const toggleAll = () => {
  collapsed.value = allCollapsed.value ? new Set() : new Set(groupKeys.value);
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur();
  }
};
</script>
