<template>
  <div v-if="ready" data-testid="side-menu">
    <div class="breadcrumbs">
      <ul>
        <li><a @click.prevent="setHost(null)" class="link-primary">Hosts</a></li>
        <li v-if="sessionHost && hosts[sessionHost]" class="cursor-default">
          {{ hosts[sessionHost].name }}
        </li>
      </ul>
    </div>
    <transition :name="sessionHost ? 'slide-left' : 'slide-right'" mode="out-in">
      <ul class="menu p-0" v-if="!sessionHost">
        <li v-for="host in config.hosts">
          <a @click.prevent="setHost(host.id)">
            <ph:computer-tower />
            {{ host.name }}
          </a>
        </li>
      </ul>
      <transition-group tag="ul" name="list" class="containers menu p-0 [&_li.menu-title]:px-0" v-else>
        <li
          v-for="item in menuItems"
          :key="isContainer(item) ? item.id : item.keyLabel"
          :class="isContainer(item) ? item.state : 'menu-title'"
          :data-testid="isContainer(item) ? null : item.keyLabel"
        >
          <popup v-if="isContainer(item)">
            <router-link
              :to="{ name: 'container-id', params: { id: item.id } }"
              active-class="active-primary"
              @click.alt.stop.prevent="store.appendActiveContainer(item)"
              :title="item.name"
            >
              <div class="truncate">
                {{ item.name }}<span class="font-light opacity-70" v-if="item.isSwarm">{{ item.swarmId }}</span>
              </div>
              <container-health :health="item.health"></container-health>
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
              <container-popup :container="item"></container-popup>
            </template>
          </popup>
          <template v-else>
            {{ $t(item.keyLabel) }}
          </template>
        </li>
      </transition-group>
    </transition>
  </div>
  <div role="status" class="flex animate-pulse flex-col gap-4" v-else>
    <div class="h-3 w-full rounded-full bg-base-content/50 opacity-50" v-for="_ in 9"></div>
    <span class="sr-only">Loading...</span>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { sessionHost } from "@/composable/storage";

const store = useContainerStore();

const { activeContainers, visibleContainers, ready } = storeToRefs(store);

function setHost(host: string | null) {
  sessionHost.value = host;
}

const debouncedIds = debouncedRef(pinnedContainers, 200);
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

const groupedContainers = computed(() =>
  sortedContainers.value.reduce(
    (acc, item) => {
      if (debouncedIds.value.has(item.name)) {
        acc.pinned.push(item);
      } else {
        acc.unpinned.push(item);
      }
      return acc;
    },
    { pinned: [] as Container[], unpinned: [] as Container[] },
  ),
);

function isContainer(item: any): item is Container {
  return item.hasOwnProperty("image");
}

const menuItems = computed(() => {
  const pinnedLabel = { keyLabel: "label.pinned" };
  const allLabel = { keyLabel: showAllContainers.value ? "label.all-containers" : "label.running-containers" };
  if (groupedContainers.value.pinned.length > 0) {
    return [pinnedLabel, ...groupedContainers.value.pinned, allLabel, ...groupedContainers.value.unpinned];
  } else {
    return [allLabel, ...groupedContainers.value.unpinned];
  }
});

const hosts = computed(() =>
  config.hosts.reduce(
    (acc, item) => {
      acc[item.id] = item;
      return acc;
    },
    {} as Record<string, { name: string; id: string }>,
  ),
);

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

.slide-left-enter-active,
.slide-left-leave-active,
.slide-right-enter-active,
.slide-right-leave-active {
  transition: all 0.1s ease-out;
}

.slide-left-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-right-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}

.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.slide-right-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.list-move,
.list-enter-active,
.list-leave-active {
  transition: all 0.19s ease;
}

.list-enter-from,
.list-leave-to {
  opacity: 0;
  transform: translateX(30px);
}

.list-leave-active {
  position: absolute;
}
</style>
