<template>
  <div class="@container flex flex-1 items-center gap-1.5 truncate md:gap-2">
    <label class="swap swap-rotate size-4">
      <input type="checkbox" v-model="pinned" />
      <carbon:star-filled class="swap-on text-secondary" />
      <carbon:star class="swap-off" />
    </label>
    <div class="inline-flex items-center font-mono text-sm">
      <div class="breadcrumbs p-0">
        <ul>
          <li v-if="config.hosts.length > 1" class="mobile-hidden font-thin">
            {{ container.hostLabel }}
          </li>
          <li>
            <div>
              <button
                popovertarget="popover-container-list"
                class="btn btn-sm"
                style="anchor-name: --anchor-popover-container-list"
              >
                {{ container.name }} <carbon:caret-down />
              </button>
              <ul
                class="dropdown menu rounded-box bg-base-100 w-52 shadow-sm"
                popover
                id="popover-container-list"
                style="position-anchor: --anchor-popover-container-list"
              >
                <li v-for="other in otherContainers">
                  <router-link :to="{ name: '/container/[id]', params: { id: other.id } }">
                    {{ other.name }}
                  </router-link>
                </li>
              </ul>
            </div>
          </li>
        </ul>
      </div>
    </div>
    <ContainerHealth :health="container.health" v-if="container.health" />
    <Tag class="mobile-hidden hidden font-mono @3xl:block" size="small">
      {{ container.image.replace(/@sha.*/, "") }}
    </Tag>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";

const { container } = defineProps<{ container: Container }>();
const pinned = computed({
  get: () => pinnedContainers.value.has(container.name),
  set: (value) => {
    if (value) {
      pinnedContainers.value.add(container.name);
    } else {
      pinnedContainers.value.delete(container.name);
    }
  },
});
const store = useContainerStore();
const { containers: allContainers } = storeToRefs(store);

const otherContainers = computed(() =>
  [...allContainers.value.filter((c) => c.name === container.name)].sort((a, b) => +b.created - +a.created),
);
</script>
