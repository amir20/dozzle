<template>
  <NavHeader :title="$t('label.hosts')" :back="selectedHost ? $t('label.hosts') : undefined" @back="setHost(null)">
    <template #title v-if="selectedHost">
      <HostIcon :type="selectedHost.type" class="text-base-content/50 size-4 shrink-0" />
      <span class="truncate text-[0.9375rem] font-medium">{{ selectedHost.name }}</span>
    </template>

    <template #actions>
      <NavMergeLink v-if="selectedHost" :to="{ name: '/host/[id]', params: { id: selectedHost.id } }" />
      <NavOverflow>
        <button type="button" class="nav-menu-item" @click="toggleShowAllContainers()">
          <mdi:check class="size-4 shrink-0" v-if="showAllContainers" />
          <span class="size-4 shrink-0" v-else></span>
          {{ $t("label.show-all-containers") }}
        </button>
        <button type="button" class="nav-menu-item" v-if="hasCollapsible" @click="collapseAll()">
          <material-symbols-light:expand-all class="size-4 shrink-0 opacity-60" v-if="allCollapsed" />
          <material-symbols-light:collapse-all class="size-4 shrink-0 opacity-60" v-else />
          {{ allCollapsed ? $t("label.expand-all") : $t("label.collapse-all") }}
        </button>
      </NavOverflow>
    </template>
  </NavHeader>

  <SlideTransition :slide-right="!!sessionHost">
    <template #left>
      <ul class="space-y-px">
        <template v-if="!hasHostGroups">
          <HostNavItem v-for="host in hosts" :key="host.id" :host="host" @click="setHost(host.id)" />
        </template>

        <template v-else v-for="[groupName, groupHosts] in groupedHostEntries" :key="groupName || '__ungrouped__'">
          <NavGroup
            v-if="groupName"
            :label="groupName"
            :count="groupHosts.length"
            :icon="Folder"
            :open="!collapsedHostGroups.has(groupName)"
            @update:open="setCollapsed(collapsedHostGroups, groupName, $event)"
          >
            <template #actions>
              <NavMergeLink :to="{ name: '/host-group/[name]', params: { name: groupName } }" />
            </template>
            <HostNavItem v-for="host in groupHosts" :key="host.id" :host="host" @click="setHost(host.id)" />
          </NavGroup>

          <template v-else>
            <HostNavItem v-for="host in groupHosts" :key="host.id" :host="host" @click="setHost(host.id)" />
          </template>
        </template>
      </ul>
    </template>

    <template #right>
      <ul class="containers space-y-1">
        <NavGroup
          v-for="{ label, containers, icon } in menuItems"
          :key="label"
          :label="label.startsWith('label.') ? $t(label) : label"
          :count="containers.length"
          :icon="icon"
          :open="!collapsedGroups.has(label)"
          @update:open="setCollapsed(collapsedGroups, label, $event)"
        >
          <template #actions>
            <NavMergeLink :to="{ name: '/merged/[ids]', params: { ids: containers.map(({ id }) => id).join(',') } }" />
          </template>

          <Popup v-for="item in containers" :key="item.id">
            <NavItem
              :to="{ name: '/container/[id]', params: { id: item.id } }"
              :label="item.name"
              :title="item.name"
              :class="[item.state, { 'highlight-new': item.isNew, 'is-merged': isMerged && isStreaming(item.id) }]"
              @click.alt.stop.prevent="pinnedStore.pinContainer(item)"
              @animationend="item.isNew = false"
            >
              <template #icon>
                <svg-spinners:ring-resize v-if="item.isNew" class="text-secondary size-4" />
                <ContainerIcon v-else :state="item.state" :health="item.health" :slug="item.icon" class="size-5" />
              </template>
              <!-- The tint alone reads as "selected"; the arrows say why several rows
                   are selected at once. -->
              <template #trailing v-if="isMerged && isStreaming(item.id)">
                <ph:arrows-merge class="size-3.5" />
              </template>
            </NavItem>
            <template #content>
              <ContainerPopup :container="item" />
            </template>
          </Popup>
        </NavGroup>
      </ul>
    </template>
  </SlideTransition>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { sessionHost } from "@/composable/app/storage";
import { useStreamedContainers } from "@/composable/containers/streamedContainers";
import { showAllContainers, groupContainers } from "@/stores/settings";

import Pin from "~icons/ph/map-pin-simple";
import Stack from "~icons/ph/stack";
import Folder from "~icons/ph/folders";
import Containers from "~icons/octicon/container-24";

const containerStore = useContainerStore();
const { visibleContainers } = storeToRefs(containerStore);

const pinnedStore = usePinnedLogsStore();

const { isStreaming, isMerged } = useStreamedContainers();

const { hosts } = useHosts();

const setHost = (host: string | null) => (sessionHost.value = host);

const selectedHost = computed(() => (sessionHost.value ? hosts.value[sessionHost.value] : undefined));

const hasHostGroups = computed(() => Object.values(hosts.value).some((h) => h.group));

const groupedHostEntries = computed(() => {
  const groups: Record<string, (typeof hosts.value)[string][]> = {};
  const ungrouped: (typeof hosts.value)[string][] = [];

  for (const host of Object.values(hosts.value)) {
    if (host.group) {
      groups[host.group] ||= [];
      groups[host.group].push(host);
    } else {
      ungrouped.push(host);
    }
  }

  const entries = Object.entries(groups).sort(([a], [b]) => a.localeCompare(b)) as [string, typeof ungrouped][];
  if (ungrouped.length > 0) {
    entries.push(["", ungrouped]);
  }
  return entries;
});

const collapsedGroups = useProfileStorage("collapsedGroups", new Set<string>());
const collapsedHostGroups = useProfileStorage("collapsedHostGroups", new Set<string>());

// Takes the set itself rather than its ref: the template hands over the unwrapped
// (still reactive) value, and both call sites mutate the same object either way.
const setCollapsed = (collapsed: Set<string>, key: string, open: boolean) => {
  if (open) {
    collapsed.delete(key);
  } else {
    collapsed.add(key);
  }
};

const hasCollapsible = computed(
  () => menuItems.value.length > 0 || groupedHostEntries.value.some(([groupName]) => groupName),
);

const allCollapsed = computed(() => {
  const containerGroups = menuItems.value;
  const hostGroups = groupedHostEntries.value.filter(([groupName]) => groupName);
  if (containerGroups.length === 0 && hostGroups.length === 0) return false;
  return (
    containerGroups.every(({ label }) => collapsedGroups.value.has(label)) &&
    hostGroups.every(([groupName]) => collapsedHostGroups.value.has(groupName))
  );
});

const collapseAll = () => {
  if (allCollapsed.value) {
    menuItems.value.forEach(({ label }) => collapsedGroups.value.delete(label));
    groupedHostEntries.value.forEach(([groupName]) => {
      if (groupName) collapsedHostGroups.value.delete(groupName);
    });
  } else {
    menuItems.value.forEach(({ label }) => collapsedGroups.value.add(label));
    groupedHostEntries.value.forEach(([groupName]) => {
      if (groupName) collapsedHostGroups.value.add(groupName);
    });
  }
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur();
  }
};

const debouncedPinnedContainers = debouncedRef(pinnedContainers, 200);
const sortedContainers = computed(() =>
  visibleContainers.value.filter((c) => c.host === sessionHost.value).sort(sorter),
);

const sorter = (a: Container, b: Container) => {
  if (a.state === "running" && b.state !== "running") {
    return -1;
  } else if (a.state !== "running" && b.state === "running") {
    return 1;
  } else {
    return a.name.localeCompare(b.name);
  }
};

const menuItems = computed(() => {
  const namespaced: Record<string, Container[]> = {};
  const pinned = [];
  const singular = [];

  for (const item of sortedContainers.value) {
    const namespace = item.namespace;
    if (debouncedPinnedContainers.value?.has(item.name)) {
      pinned.push(item);
    } else if (namespace) {
      namespaced[namespace] ||= [];
      namespaced[namespace].push(item);
    } else {
      singular.push(item);
    }
  }

  const items = [];
  if (pinned.length) {
    items.push({ label: "label.pinned", containers: pinned, icon: Pin });
  }
  for (const [label, containers] of Object.entries(namespaced).sort(([a], [b]) => a.localeCompare(b))) {
    const shouldGroup =
      groupContainers.value === "always" || (groupContainers.value === "at-least-2" && containers.length > 1);

    if (shouldGroup) {
      items.push({ label, containers, icon: Stack });
    } else {
      for (const container of containers) {
        singular.push(container);
      }
    }
  }

  singular.sort(sorter);

  if (singular.length) {
    items.push({
      label: showAllContainers.value ? "label.all-containers" : "label.running-containers",
      containers: singular,
      icon: Containers,
    });
  }

  return items;
});

const route = useRoute("/container/[id]");

watch(
  [() => route.name, () => route.params.id],
  ([name, id]) => {
    if (name === "/container/[id]") {
      const container = containerStore.findContainerById(id as string);
      if (container) {
        setHost(container.host);
      }
    }
  },
  { immediate: true },
);

const toggleShowAllContainers = () => (showAllContainers.value = !showAllContainers.value);
</script>

<style scoped>
@reference "@/main.css";

.containers :deep(.exited) {
  @apply opacity-60;
}

.containers :deep(.deleted) {
  @apply hidden;
}

.containers :deep(.highlight-new) {
  animation: highlight-fade 3s ease-out;
}

@keyframes highlight-fade {
  from {
    background-color: oklch(from var(--color-secondary) l c h / 0.25);
  }
  to {
    background-color: transparent;
  }
}
</style>
