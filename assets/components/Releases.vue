<template>
  <ul class="space-y-4 p-2">
    <li v-for="release in releases">
      <div class="flex items-center justify-between">
        <h3 class="text-lg font-bold">{{ release.name }}</h3>
        <tag class="bg-red px-1 py-1 text-xs" v-if="release.tag === latest?.tag">Latest</tag>
      </div>
      <div class="text-sm">Released <distance-time :date="new Date(release.createdAt)" /></div>
    </li>
  </ul>
</template>

<script setup lang="ts">
const { releases, latest } = useReleases();
</script>

<style scoped></style>
