<template>
  <div class="breadcrumbs">
    <ul>
      <li><a @click.prevent="setHost(null)" class="link-primary">Hosts</a></li>
      <li v-if="sessionHost && hosts[sessionHost]" class="cursor-default">
        {{ hosts[sessionHost].name }}
      </li>
    </ul>
  </div>

  <SlideTransition :slide-right="!!sessionHost">
    <template #left>
      <ul class="menu p-0">
        <li v-for="host in hosts" :key="host.id">
          <a @click.prevent="setHost(host.id)" :class="{ 'pointer-events-none text-base-content/50': !host.available }">
            <ph:computer-tower />
            {{ host.name }}
            <span class="badge badge-error badge-xs p-1.5" v-if="!host.available">offline</span>
          </a>
        </li>
      </ul>
    </template>
    <template #right>
      <ul class="containers menu p-0 [&_li.menu-title]:px-0">
        <li v-for="{ label, containers, icon } in menuItems" :key="label">
          <!-- @vue-ignore -->
          <details :open="!collapsedGroups.has(label)" @toggle="updateCollapsedGroups($event, label)">
            <summary class="font-light text-base-content/80">
              <component :is="icon" />
              {{ label.startsWith("label.") ? $t(label) : label }}

              <router-link
                :to="{ name: 'stack-name', params: { name: label } }"
                class="btn btn-info btn-xs"
                v-if="!label.startsWith('label.')"
              >
                all
              </router-link>
            </summary>
            <ul>
              <li v-for="item in containers" :class="item.state" :key="item.id">
                <popup>
                  <router-link
                    :to="{ name: 'container-id', params: { id: item.id } }"
                    active-class="active-primary"
                    @click.alt.stop.prevent="store.appendActiveContainer(item)"
                    :title="item.name"
                  >
                    <div class="truncate">
                      {{ item.name }}<span class="font-light opacity-70" v-if="item.isSwarm">{{ item.swarmId }}</span>
                    </div>
                    <ContainerHealth :health="item.health" />
                    <span
                      class="pin"
                      @click.stop.prevent="store.appendActiveContainer(item)"
                      v-show="!activeContainersById[item.id]"
                      :title="$t('tooltip.pin-column')"
                    >
                      <cil:columns />
                    </span>
                  </router-link>
                  <template #content>
                    <ContainerPopup :container="item" />
                  </template>
                </popup>
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

// @ts-ignore
import Pin from "~icons/ph/map-pin-simple";
// @ts-ignore
import Stack from "~icons/ph/stack";
// @ts-ignore
import Containers from "~icons/octicon/container-24";

const store = useContainerStore();

const { activeContainers, visibleContainers } = storeToRefs(store);
const { hosts } = useHosts();

const setHost = (host: string | null) => (sessionHost.value = host);

const collapsedGroups = useProfileStorage("collapsedGroups", new Set<string>());
const updateCollapsedGroups = (event: Event, label: string) => {
  const details = event.target as HTMLDetailsElement;
  if (details.open) {
    collapsedGroups.value.delete(label);
  } else {
    collapsedGroups.value.add(label);
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
    const namespace = item.labels["com.docker.stack.namespace"] ?? item.labels["com.docker.compose.project"];
    if (debouncedPinnedContainers.value.has(item.name)) {
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
    if (containers.length > 1) {
      items.push({ label, containers, icon: Stack });
    } else {
      singular.push(containers[0]);
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

const activeContainersById = computed(() =>
  activeContainers.value.reduce(
    (acc, item) => {
      acc[item.id] = item;
      return acc;
    },
    {} as Record<string, Container>,
  ),
);
</script>
<style scoped lang="postcss">
.menu {
  @apply text-[0.95rem];
}
.containers a {
  @apply auto-cols-[auto_max-content_max-content];
  .pin {
    display: none;

    &:hover {
      @apply text-secondary;
    }
  }

  &:hover {
    .pin {
      display: inline-block;
    }
  }
}
li.exited {
  @apply opacity-50;
}
</style>
