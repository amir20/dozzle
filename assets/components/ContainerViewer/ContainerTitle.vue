<template>
  <div class="flex flex-1 gap-1.5 truncate @container md:gap-2">
    <label class="swap swap-rotate size-4">
      <input type="checkbox" v-model="pinned" />
      <carbon:star-filled class="swap-on text-secondary" />
      <carbon:star class="swap-off" />
    </label>
    <div class="inline-flex font-mono text-sm">
      <div v-if="config.hosts.length > 1" class="mobile-hidden font-thin">
        {{ container.hostLabel }}<span class="mx-2">/</span>
      </div>
      <div class="font-semibold">{{ container.name }}</div>
      <div
        class="mobile-hidden max-w-[1.5em] truncate transition-[max-width] hover:max-w-[400px]"
        v-if="container.isSwarm"
      >
        .{{ container.swarmId }}
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
</script>
