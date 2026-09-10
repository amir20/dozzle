<template>
  <NavHeader :title="$t('label.group-menu')" />

  <ul class="space-y-px" v-if="customGroups.length > 0">
    <NavItem
      v-for="group in customGroups"
      :key="group.name"
      :to="{ name: '/group/[name]', params: { name: group.name } }"
      :label="group.name"
      :title="group.name"
    >
      <template #icon><ph:bounding-box-fill class="size-4 opacity-70" /></template>
    </NavItem>
  </ul>
</template>

<script lang="ts" setup>
const store = useSwarmStore();

const { customGroups } = storeToRefs(store);
</script>
