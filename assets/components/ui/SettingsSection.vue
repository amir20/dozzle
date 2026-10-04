<template>
  <!-- One titled block of a settings page: the uppercase heading, an optional line
       under it, then the content. The actions slot sits level with the heading. -->
  <section class="flex flex-col gap-4">
    <div class="flex items-end justify-between gap-3">
      <div class="min-w-0">
        <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ title }}</h2>
        <p v-if="desc" class="text-base-content/60 mt-1 text-sm">{{ desc }}</p>
      </div>
      <slot name="actions" />
    </div>
    <slot />
  </section>
</template>

<script lang="ts" setup>
defineProps<{ title: string; desc?: string }>();
</script>
