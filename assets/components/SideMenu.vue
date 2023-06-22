<template>
  <div v-if="ready">
    <nav class="breadcrumb" aria-label="breadcrumbs">
      <ul v-if="sessionHost">
        <li>
          <a href="#" @click.prevent="setHost(null)">{{ sessionHost }}</a>
        </li>
        <li class="is-active">
          <a href="#" aria-current="page">{{ $t("label.containers") }}</a>
        </li>
      </ul>
      <ul v-else>
        <li>Hosts</li>
      </ul>
    </nav>
    <transition :name="sessionHost ? 'slide-left' : 'slide-right'" mode="out-in">
      <ul class="menu-list" v-if="!sessionHost">
        <li v-for="host in config.hosts">
          <a @click.prevent="setHost(host)">{{ host }}</a>
        </li>
      </ul>
      <ul class="menu-list" v-else>
        <li v-for="item in sortedContainers" :key="item.id" :class="item.state">
          <popup>
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
      </ul>
    </transition>
  </div>
  <ul class="menu-list is-hidden-mobile loading" v-else>
    <li v-for="index in 7" class="my-4"><o-skeleton animated size="large" :key="index"></o-skeleton></li>
  </ul>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { sessionHost } from "@/composables/storage";

const store = useContainerStore();

const { activeContainers, visibleContainers, ready } = storeToRefs(store);

function setHost(host: string | null) {
  sessionHost.value = host;
}

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
    })
);

const activeContainersById = computed(() =>
  activeContainers.value.reduce((acc, item) => {
    acc[item.id] = item;
    return acc;
  }, {} as Record<string, Container>)
);
</script>
<style scoped lang="scss">
.has-light-opacity {
  opacity: 0.5;
}

.loading {
  opacity: 0.5;
}

li.exited a {
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
  transition: all 0.2s ease-out;
}

.slide-left-enter-from {
  opacity: 0;
  transform: translateX(300px);
}

.slide-right-enter-from {
  opacity: 0;
  transform: translateX(-240px);
}

.slide-left-leave-to {
  opacity: 0;
  transform: translateX(-240px);
}

.slide-right-leave-to {
  opacity: 0;
  transform: translateX(240px);
}
</style>
