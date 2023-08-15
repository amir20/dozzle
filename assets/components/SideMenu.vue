<template>
  <div v-if="ready">
    <nav class="breadcrumb menu-label" aria-label="breadcrumbs">
      <ul v-if="sessionHost">
        <li>
          <a href="#" @click.prevent="setHost(null)">{{ hosts[sessionHost].name }}</a>
        </li>
      </ul>
      <ul v-else>
        <li>Hosts</li>
      </ul>
    </nav>
    <transition :name="sessionHost ? 'slide-left' : 'slide-right'" mode="out-in">
      <ul class="menu-list" v-if="!sessionHost">
        <li v-for="host in config.hosts">
          <a @click.prevent="setHost(host.id)">{{ host.name }}</a>
        </li>
      </ul>
      <transition-group tag="ul" name="list" class="menu-list" v-else>
        <li v-for="item in menuItems" :key="item.id" :class="item.state" :data-label="item.id">
          <div class="menu-label mt-4 mb-3" v-if="isLabel(item)">
            {{ item.label }}
          </div>
          <popup v-else>
            <router-link
              :to="{ name: 'container-id', params: { id: item.id } }"
              active-class="is-active"
              :title="item.name"
            >
              <div class="container is-flex is-align-items-center">
                <div class="is-flex-grow-1 is-ellipsis">
                  <span>{{ item.name }}</span
                  ><span class="has-text-weight-light has-light-opacity" v-if="item.isSwarm">{{ item.swarmId }}</span>
                </div>
                <div class="is-flex-shrink-1 is-flex icons">
                  <div
                    class="icon is-small pin"
                    @click.stop.prevent="store.appendActiveContainer(item)"
                    v-show="!activeContainersById[item.id]"
                    :title="$t('tooltip.pin-column')"
                  >
                    <cil:columns />
                  </div>

                  <container-health :health="item.health"></container-health>
                </div>
              </div>
            </router-link>
            <template #content>
              <container-popup :container="item"></container-popup>
            </template>
          </popup>
        </li>
      </transition-group>
    </transition>
  </div>
  <ul class="menu-list is-hidden-mobile has-light-opacity" v-else>
    <li v-for="index in 7" class="my-4"><o-skeleton animated size="large" :key="index"></o-skeleton></li>
  </ul>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { sessionHost } from "@/composables/storage";

const { t } = useI18n();

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
      if (debouncedIds.value.has(item.storageKey)) {
        acc.pinned.push(item);
      } else {
        acc.unpinned.push(item);
      }
      return acc;
    },
    { pinned: [] as Container[], unpinned: [] as Container[] },
  ),
);

type MenuLabel = { label: string; id: string; state: string };
const pinnedLabel = { label: t("label.pinned"), id: "pinned", state: "label" } as MenuLabel;
const allLabel = { label: t("label.containers"), id: "all", state: "label" } as MenuLabel;

function isLabel(item: Container | MenuLabel): item is MenuLabel {
  return (item as MenuLabel).label !== undefined;
}

const menuItems = computed(() => {
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
<style scoped lang="scss">
.has-light-opacity {
  opacity: 0.5;
}

li.exited a,
li.dead a {
  color: #777;
}

.icons {
  column-gap: 0.35em;
  align-items: baseline;
}
a {
  .pin {
    display: none;

    &:hover {
      color: var(--secondary-color);
    }
  }

  &:hover {
    .pin {
      display: block;
    }
  }
}

.slide-left-enter-active,
.slide-left-leave-active,
.slide-right-enter-active,
.slide-right-leave-active {
  transition: all 0.1s ease-out;
}

.slide-left-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.slide-right-enter-from {
  opacity: 0;
  transform: translateX(-100%);
}

.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-100%);
}

.slide-right-leave-to {
  opacity: 0;
  transform: translateX(100%);
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
