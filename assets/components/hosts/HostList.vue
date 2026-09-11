<template>
  <ul class="grid gap-3 md:grid-cols-[repeat(auto-fit,minmax(min(100%,30rem),1fr))]">
    <!-- The card lays itself out against this, not the viewport: how wide a card
         gets depends on how many columns the grid settled on. -->
    <li class="@container" v-for="host in hosts" :key="host.id">
      <HostCard :host="host" />
    </li>
  </ul>
</template>

<script setup lang="ts">
const { hosts } = useHosts();
</script>
