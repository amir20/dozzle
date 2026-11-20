<template>
  <Popover
    :placement="placement"
    panel-class="rounded-box border-base-content/20 bg-base-200 min-w-52 border p-2 shadow-sm"
    :close-on-select="false"
    @open="$emit('opened')"
    @close="$emit('closed')"
  >
    <template #trigger>
      <label tabindex="0" :class="triggerClass">
        <slot name="trigger"></slot>
      </label>
    </template>
    <template #default="{ close }">
      <slot name="content" :close="close"></slot>
    </template>
  </Popover>
</template>

<script setup lang="ts">
const { placement = "bottom-start", triggerClass = "btn btn-circle btn-sm" } = defineProps<{
  placement?: PopoverPlacement;
  triggerClass?: string;
}>();

defineEmits<{ opened: []; closed: [] }>();
</script>
