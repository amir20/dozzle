<template>
  <div class="flex items-center gap-2">
    <container-health :health="container.health" v-if="container.health"></container-health>
    <div class="name inline-flex truncate">
      <div v-if="config.hosts.length > 1" class="host has-text-weight-light is-hidden-mobile">
        {{ container.hostLabel }}<span class="has-text-weight-light mx-2">/</span>
      </div>
      <div>{{ container.name }}</div>
      <div
        class="mobile-hidden max-w-[1.5em] truncate transition-[max-width] hover:max-w-[400px]"
        v-if="container.isSwarm"
      >
        {{ container.swarmId }}
      </div>
    </div>
    <tag class="mobile-hidden">{{ container.image.replace(/@sha.*/, "") }}</tag>
    <div class="cursor-pointer" @click="togglePinnedContainer(container.storageKey)">
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

<style scoped></style>
