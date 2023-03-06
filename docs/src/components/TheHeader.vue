<script setup lang="ts">
const emit = defineEmits(['menu-click'])
</script>

<template>
  <header py-2 lg:px-4 main-bg mx-auto>
    <nav flex my-3 gap-x-4 justify-end items-center>
      <a v-if="isMobile" cursor-pointer text-2xl i-mdi-menu @click="emit('menu-click')" />
      <h1 font-playfair mr-auto text-4xl dark:text-brand>
        <a href="/">Dozzle</a>
      </h1>
      <slot />

      <a
        icon-btn
        i-mdi-docker
        text-xl
        target="_blank"
        rel="noreferrer"
        href="https://hub.docker.com/r/amir20/dozzle/"
      />
      <a
        icon-btn
        i-mdi-github
        text-xl
        rel="noreferrer"
        href="https://github.com/amir20/dozzle"
        target="_blank"
        title="Dozzle GitHub"
      />
    </nav>
  </header>
</template>
