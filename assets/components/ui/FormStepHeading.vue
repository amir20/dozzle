<template>
  <div class="mb-2 flex items-center gap-2">
    <span
      class="bg-base-content/10 text-base-content/60 flex size-5 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
      aria-hidden="true"
    >
      {{ step }}
    </span>
    <h3 class="text-base-content/70 text-sm font-semibold tracking-wide uppercase">
      {{ title }}
      <!-- Muted on purpose: a column of red asterisks reads as a column of errors. -->
      <span v-if="required" class="text-base-content/35" :title="$t('notifications.alert-form.required')">*</span>
    </h3>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ step: number; title: string; required?: boolean }>();
</script>
