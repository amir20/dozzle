<template>
  <!--
    The in-page counterpart to ToastModal: a neutral panel in the same family as the
    dropdowns and drawers, with severity carried by the icon alone. daisyUI's `alert`
    paints the whole block in the severity color, which at drawer or page width shouts
    over everything around it and drops the text to whatever contrast the tint allows.
  -->
  <div class="rounded-box border-base-content/10 bg-base-200 flex items-start gap-2.5 border p-3">
    <div class="mt-0.5 shrink-0" :class="accent[type]">
      <mdi:check-circle-outline v-if="type === 'success'" class="size-4" />
      <mdi:alert-circle-outline v-else-if="type === 'error'" class="size-4" />
      <mdi:alert-outline v-else-if="type === 'warning'" class="size-4" />
      <mdi:information-outline v-else class="size-4" />
    </div>
    <div class="min-w-0 flex-1 text-sm leading-relaxed">
      <slot />
    </div>
    <!-- Actions sit on the end of the row rather than under it: every notice that has
         them here is a one-line question with one or two short answers. -->
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ type: "info" | "success" | "warning" | "error" }>();

const accent = {
  info: "text-info",
  success: "text-success",
  warning: "text-warning",
  error: "text-error",
} as const;
</script>
