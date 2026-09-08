<template>
  <div class="mb-2 flex h-7 shrink-0 items-center gap-1">
    <button type="button" class="icon-btn nav-btn" v-if="back" :title="back" :aria-label="back" @click="$emit('back')">
      <mdi:chevron-left class="size-4" />
    </button>

    <div class="flex min-w-0 flex-1 items-center gap-1.5">
      <slot name="title">
        <span class="truncate text-sm font-medium">{{ title }}</span>
      </slot>
    </div>

    <div class="flex shrink-0 items-center gap-0.5">
      <slot name="actions" />
    </div>
  </div>
</template>

<script lang="ts" setup>
const { title, back } = defineProps<{
  title?: string;
  /** Label for the back affordance. Omitted means there is nowhere to go back to. */
  back?: string;
}>();

defineEmits<{ back: [] }>();
</script>
