<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <div class="column is-ellipsis">
      <container-health :health="container.health" v-if="container.health"></container-health>
      <span class="name">
        {{ container.name }}<span v-if="container.isSwarm" class="swarm-id">{{ container.swarmId }}</span>
      </span>
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
  .swarm-id {
    display: none;
  }

  &:hover {
    .swarm-id {
      display: inline;
    }
  }
}
</style>
