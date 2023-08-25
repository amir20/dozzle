<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <div class="column is-ellipsis">
      <container-health :health="container.health" v-if="container.health"></container-health>
      <div class="name">
        <span v-if="config.hosts.length > 1" class="host has-text-weight-light is-hidden-mobile"
          >{{ container.hostLabel }}<span class="has-text-weight-light mx-2">/</span></span
        ><span class="">{{ container.name }}</span
        ><span v-if="container.isSwarm" class="swarm-id is-hidden-mobile is-ellipsis">{{ container.swarmId }}</span>
      </div>
      <tag class="is-hidden-mobile">{{ container.image.replace(/@sha.*/, "") }}</tag>
      <span class="icon is-clickable" @click="togglePinnedContainer(container.storageKey)">
        <carbon:star-filled v-if="pinned" />
        <carbon:star v-else />
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const container = inject("container") as ComputedRef<Container>;
const pinned = computed(() => pinnedContainers.value.has(container.value.storageKey));
</script>

<style lang="scss" scoped>
.icon {
  vertical-align: middle;
}

.name {
  display: inline-flex;
  .swarm-id {
    max-width: 1.5em;
    display: inline-block;
    overflow: hidden;
    white-space: nowrap;
    transition: max-width 0.2s ease-in-out;
    will-change: max-width;
  }

  &:hover {
    .swarm-id {
      max-width: 400px;
    }
  }
}
</style>
