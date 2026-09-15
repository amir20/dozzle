<template>
  <!--
    A route whose subject is gone. The EmptyState centered in the room the view would
    have filled, with the one way out, so a stale link reads as quiet rather than broken.
  -->
  <div class="flex min-h-screen items-center justify-center">
    <EmptyState :title :hint>
      <template #icon><slot name="icon" /></template>
      <template #actions>
        <RouterLink to="/" class="btn btn-sm">{{ $t("button.back-to-dashboard") }}</RouterLink>
      </template>
    </EmptyState>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ title: string; hint?: string }>();
</script>
