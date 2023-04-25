<template>
  <div class="columns is-marginless has-text-weight-bold is-family-monospace">
    <span class="column is-ellipsis">
      <span class="icon is-small health" :health="container.health" v-if="container.health" :title="container.health">
        <cil:check-circle v-if="container.health == 'healthy'" />
        <cil:x-circle v-else-if="container.health == 'unhealthy'" />
        <cil:circle v-else />
      </span>
      {{ container.name }}<span v-if="container.isSwarm">.{{ container.swarmId }}</span>
      <span class="tag is-dark">{{ container.image.replace(/@sha.*/, "") }}</span>
    </span>
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

.health {
  &[health="unhealthy"] {
    color: var(--red-color);
  }

  &[health="healthy"] {
    color: var(--green-color);
  }
}
</style>
