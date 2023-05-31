<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <div class="column is-narrow" v-if="container.health">
      <container-health :health="container.health"></container-health>
    </div>
    <div class="column is-ellipsis">
      {{ container.name }}<span v-if="container.isSwarm">{{ container.swarmId }}</span>
      <tag class="is-hidden-mobile">{{ container.image.replace(/@sha.*/, "") }}</tag>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { type ComputedRef } from "vue";

const container = inject("container") as ComputedRef<Container>;
</script>

<style lang="scss" scoped>
.icon {
  vertical-align: middle;
}
</style>
