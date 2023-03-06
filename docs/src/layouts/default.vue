<script setup lang="ts">
import { useHead } from '@vueuse/head'
useHead({
  title: 'Dozzle - Log viewer for Docker',
})
</script>

<template>
  <the-header>
    <router-link to="/guide/what-is-dozzle">
      Guide
    </router-link>
  </the-header>
  <RouterView />
</template>
