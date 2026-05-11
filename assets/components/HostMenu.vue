<template>
  <div class="flex items-center">
    <div class="breadcrumbs flex-1">
      <ul>
        <li>
          <a @click.prevent="setHost(null)" class="link-primary">{{ $t("label.hosts") }}</a>
        </li>
        <li v-if="sessionHost && hosts[sessionHost]" class="cursor-default">
          <router-link
            :to="{
              name: '/host/[id]',
              params: { id: hosts[sessionHost].id },
            }"
            class="btn btn-outline btn-primary btn-xs"
            active-class="btn-active"
            :title="$t('tooltip.merge-all')"
          >
            <ph:arrows-merge />
            {{ hosts[sessionHost].name }}
          </router-link>
        </li>
      </ul>
    </div>
    <div class="flex-none">
      <div class="dropdown dropdown-end dropdown-hover">
        <label tabindex="0" class="btn btn-square btn-ghost btn-sm">
          <ion:ellipsis-vertical />
        </label>
        <ul
          tabindex="0"
          class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 w-52 border p-1 shadow-sm"
        >
          <li>
            <a class="text-sm capitalize" @click="toggleShowAllContainers()">
              <mdi:check class="w-4" v-if="showAllContainers" />
              <div v-else class="w-4"></div>
              {{ $t("label.show-all-containers") }}
            </a>
            <a v-if="hasCollapsible" class="text-sm capitalize" @click="collapseAll()">
              <material-symbols-light:expand-all class="w-4" v-if="allCollapsed" />
              <material-symbols-light:collapse-all class="w-4" v-else />
              {{ allCollapsed ? $t("label.expand-all") : $t("label.collapse-all") }}
            </a>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <SlideTransition :slide-right="!!sessionHost">
    <template #left>
      <ul class="menu p-0">
        <template v-if="!hasHostGroups">
          <li v-for="host in hosts" :key="host.id">
            <a
              @click.prevent="setHost(host.id)"
              :class="{ 'text-base-content/50 pointer-events-none': !host.available }"
            >
              <HostIcon :type="host.type" />
              {{ host.name }}
              <span class="badge badge-error badge-xs p-1.5" v-if="!host.available">offline</span>
            </a>
          </li>
        </template>
        <template v-else v-for="[groupName, groupHosts] in groupedHostEntries" :key="groupName || '__ungrouped__'">
          <li v-if="groupName" class="host-group">
            <details :open="!collapsedHostGroups.has(groupName)" @toggle="updateCollapsedHostGroups($event, groupName)">
              <summary class="host-group-summary">
                <span class="truncate">{{ groupName }}</span>
                <router-link
                  :to="{ name: '/host-group/[name]', params: { name: groupName } }"
                  class="btn btn-square btn-outline btn-primary btn-xs"
                  :title="$t('tooltip.merge-all')"
                  @click.stop
                >
                  <ph:arrows-merge />
                </router-link>
                <button
                  v-if="!collapsedHostGroups.has(groupName)"
                  type="button"
                  class="btn btn-square btn-outline btn-primary btn-xs"
                  :title="$t('label.collapse-group')"
                  @click.stop.prevent="collapseHostGroup(groupName)"
                >
                  <material-symbols-light:collapse-all />
                </button>
              </summary>
              <ul>
                <li v-for="host in groupHosts" :key="host.id">
                  <a
                    @click.prevent="setHost(host.id)"
                    :class="{ 'text-base-content/50 pointer-events-none': !host.available }"
                  >
                    <HostIcon :type="host.type" />
                    {{ host.name }}
                    <span class="badge badge-error badge-xs p-1.5" v-if="!host.available">offline</span>
                  </a>
                </li>
              </ul>
            </details>
          </li>
          <template v-else>
            <li v-for="host in groupHosts" :key="host.id">
              <a
                @click.prevent="setHost(host.id)"
                :class="{ 'text-base-content/50 pointer-events-none': !host.available }"
              >
                <HostIcon :type="host.type" />
                {{ host.name }}
                <span class="badge badge-error badge-xs p-1.5" v-if="!host.available">offline</span>
              </a>
            </li>
          </template>
        </template>
      </ul>
    </template>
    <template #right>
      <ul class="containers menu w-full p-0 [&_li.menu-title]:px-0">
        <li v-for="{ label, containers, icon } in menuItems" :key="label">
          <details :open="!collapsedGroups.has(label)" @toggle="updateCollapsedGroups($event, label)">
            <summary class="text-base-content/80 font-light">
              <component :is="icon" />
              {{ label.startsWith("label.") ? $t(label) : label }} ({{ containers.length }})

              <router-link
                :to="{
                  name: '/merged/[ids]',
                  params: { ids: containers.map(({ id }) => id).join(',') },
                }"
                class="btn btn-square btn-outline btn-primary btn-xs"
                active-class="btn-active"
                :title="$t('tooltip.merge-all')"
              >
                <ph:arrows-merge />
              </router-link>
            </summary>
            <ul>
              <li
                v-for="item in containers"
                :class="[item.state, { 'highlight-new': item.isNew }]"
                :key="item.id"
                @animationend="item.isNew = false"
              >
                <Popup>
                  <router-link
                    :to="{ name: '/container/[id]', params: { id: item.id } }"
                    active-class="menu-active"
                    @click.alt.stop.prevent="pinnedStore.pinContainer(item)"
                    :title="item.name"
                    class="group auto-cols-[content_max_auto_max-content_max-content]"
                  >
                    <svg-spinners:ring-resize v-if="item.isNew" class="text-secondary w-2" />
                    <div
                      v-else
                      class="status data-[state=exited]:status-error data-[state=running]:status-success"
                      :data-state="item.state"
                    ></div>
                    <div class="truncate">
                      {{ item.name }}
                    </div>
                    <ContainerHealth :health="item.health" />
                    <span
                      class="hover:text-secondary hidden group-hover:inline-block"
                      @click.stop.prevent="pinnedStore.pinContainer(item)"
                      v-show="!pinnedStore.isPinned(item)"
                      :title="$t('tooltip.pin-column')"
                    >
                      <cil:columns />
                    </span>
                  </router-link>
                  <template #content>
                    <ContainerPopup :container="item" />
                  </template>
                </Popup>
              </li>
            </ul>
          </details>
        </li>
      </ul>
    </template>
  </SlideTransition>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { sessionHost } from "@/composable/storage";
import { showAllContainers, groupContainers } from "@/stores/settings";

// @ts-ignore
import Pin from "~icons/ph/map-pin-simple";
// @ts-ignore
import Stack from "~icons/ph/stack";
// @ts-ignore
import Containers from "~icons/octicon/container-24";

const containerStore = useContainerStore();
const { visibleContainers } = storeToRefs(containerStore);

const pinnedStore = usePinnedLogsStore();

const { hosts } = useHosts();

const setHost = (host: string | null) => (sessionHost.value = host);

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
const updateCollapsedGroups = (event: Event, label: string) => {
  const details = event.target as HTMLDetailsElement;
  if (details.open) {
    collapsedGroups.value.delete(label);
  } else {
    collapsedGroups.value.add(label);
  }
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur();
  }
};

const updateCollapsedHostGroups = (event: Event, groupName: string) => {
  const details = event.target as HTMLDetailsElement;
  if (details.open) {
    collapsedHostGroups.value.delete(groupName);
  } else {
    collapsedHostGroups.value.add(groupName);
  }
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur();
  }
};

const collapseHostGroup = (groupName: string) => {
  collapsedHostGroups.value.add(groupName);
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur();
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
.menu {
  @apply text-[0.95rem];
}

.host-group-summary {
  display: grid;
  grid-template-columns: minmax(0, auto) max-content max-content max-content;
  align-items: center;
  justify-content: start;
  gap: 0.5rem;
  padding-left: 0;
  padding-right: 0.25rem;
  color: color-mix(in oklch, var(--color-base-content) 50%, transparent);
}

.host-group-summary::after {
  margin-left: 0.25rem;
}

li.exited {
  @apply opacity-75;
}

li.deleted {
  @apply hidden;
}

li.highlight-new {
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
