<script lang="ts" setup>
import { withBase } from "vitepress";
</script>

<template>
  <div class="mt-16 mb-4 flex items-center justify-center px-6">
    <a
      class="flex flex-col items-center gap-2 text-center text-sm text-(--vp-c-text-2) no-underline! transition-colors hover:text-(--vp-c-brand) sm:flex-row"
      href="https://www.docker.com/blog/docker-sponsored-open-source-program/"
    >
      <img :src="withBase('/icons/docker-icon.svg')" width="20" height="20" alt="" />
      Open source and part of Docker's sponsored OSS program
    </a>
  </div>
</template>
