<template>
  <ul class="grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(480px,1fr))]">
    <li v-for="host in hosts" :key="host.id">
      <HostCard :host="host" />
    </li>
  </ul>
</template>

<script setup lang="ts">
const { hosts } = useHosts();
</script>
