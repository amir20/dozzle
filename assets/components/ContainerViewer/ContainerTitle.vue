<template>
  <div class="@container flex flex-1 items-center gap-1.5 md:gap-2">
    <label class="swap swap-rotate size-4">
      <input type="checkbox" v-model="pinned" />
      <carbon:star-filled class="swap-on text-secondary" />
      <carbon:star class="swap-off" />
    </label>
    <div class="inline-flex items-center text-sm">
      <div class="breadcrumbs overflow-x-visible p-0 font-mono">
        <ul>
          <li v-if="config.hosts.length > 1" class="font-thin max-md:hidden">
            {{ container.hostLabel }}
          </li>
          <li>
            <template v-if="otherContainers.length === 0">{{ container.name }}</template>
            <div v-else>
              <div class="dropdown">
                <button tabindex="0" role="button" class="btn btn-xs md:btn-sm">
                  {{ container.name }} <carbon:caret-down />
                </button>
                <ul
                  tabindex="0"
                  class="dropdown-content menu rounded-box bg-base-100 border-base-content/20 border shadow-sm"
                >
                  <li v-for="other in otherContainers">
                    <router-link :to="{ name: '/container/[id]', params: { id: other.id } }">
                      <div
                        class="status data-[state=exited]:status-error data-[state=running]:status-success"
                        :data-state="other.state"
                      ></div>
                      <div v-if="other.isSwarm">{{ other.swarmId }}</div>
                      <div v-else>{{ other.name }}</div>
                      <div v-if="other.state === 'running'">running</div>
                      <RelativeTime :date="other.finishedAt" class="text-base-content/70 text-xs" v-else />
                    </router-link>
                  </li>
                </ul>
              </div>
            </div>
          </li>
        </ul>
      </div>
    </div>
    <ContainerHealth :health="container.health" v-if="container.health" />
    <Tag class="hidden! font-mono @xl:block!" size="small">
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
  allContainers.value
    .filter((c) => c.name === container.name && c.id !== container.id)
    .sort((a, b) => +b.created - +a.created),
);
</script>

<style scoped></style>
