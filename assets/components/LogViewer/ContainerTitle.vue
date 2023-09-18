<template>
  <div class="flex items-center gap-2">
    <container-health :health="container.health" v-if="container.health"></container-health>
    <div class="inline-flex truncate font-mono text-sm">
      <div v-if="config.hosts.length > 1" class="mobile-hidden font-thin">
        {{ container.hostLabel }}<span class="mx-2">/</span>
      </div>
      <div class="font-semibold">{{ container.name }}</div>
      <div
        class="mobile-hidden max-w-[1.5em] truncate transition-[max-width] hover:max-w-[400px]"
        v-if="container.isSwarm"
      >
        {{ container.swarmId }}
      </div>
    </div>
    <tag class="mobile-hidden font-mono" size="small">{{ container.image.replace(/@sha.*/, "") }}</tag>
    <div class="inline-flex cursor-pointer" @click="togglePinnedContainer(container.storageKey)">
      <carbon:star-filled v-if="pinned" />
      <carbon:star v-else />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const container = inject("container") as ComputedRef<Container>;
const pinned = computed(() => pinnedContainers.value.has(container.value.storageKey));
</script>
