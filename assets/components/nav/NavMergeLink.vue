<template>
  <router-link
    :to="to"
    class="icon-btn nav-btn"
    active-class="bg-primary/15 text-primary"
    :title="$t('tooltip.merge-all')"
    :aria-label="$t('tooltip.merge-all')"
    @click.stop
  >
    <ph:arrows-merge class="size-4" />
  </router-link>
</template>

<script lang="ts" setup>
import type { RouteLocationRaw } from "vue-router";

const { to } = defineProps<{ to: RouteLocationRaw }>();
</script>
