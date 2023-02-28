<script setup lang="ts">
const menu = [
  {
    name: 'What is Dozzle?',
    path: '/guide/what-is-dozzle',
  },
  {
    name: 'Getting Started',
    path: '/guide/getting-started',
  },
]
</script>

<template>
  <div>
    <ul>
      <li v-for="item in menu" :key="item.path">
        <router-link :to="item.path">
          {{ item.name }}
        </router-link>
      </li>
    </ul>
  </div>
  <RouterView />
</template>
