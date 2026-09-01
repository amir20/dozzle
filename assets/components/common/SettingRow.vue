<template>
  <!-- One row of a settings card: label, optional description, control on the right.
       Renders as a <label> for rows whose control is a single input, so clicking the
       text toggles it; rows holding a dropdown or a button group stay a <div> to avoid
       nesting several interactive elements inside one label. -->
  <component :is="tag" class="flex min-h-13 flex-wrap items-center justify-between gap-3 py-3.5 text-sm font-medium">
    <span class="min-w-0">
      <span class="inline-flex items-center gap-2">
        {{ label }}
        <slot name="label-suffix" />
      </span>
      <span v-if="description" class="text-base-content/60 mt-0.5 block text-xs font-normal">
        {{ description }}
      </span>
    </span>
    <span class="ml-auto flex flex-wrap items-center gap-2">
      <slot />
    </span>
  </component>
</template>

<script lang="ts" setup>
const { tag = "div" } = defineProps<{
  label: string;
  description?: string;
  tag?: "label" | "div";
}>();
</script>
