<template>
  <!-- One row of a settings card: label, optional description, control on the right.
       Renders as a <label> for rows whose control is a single input, so clicking the
       text toggles it; rows holding a dropdown or a button group stay a <div> to avoid
       nesting several interactive elements inside one label.

       The control never wraps below the text: on a phone that gave every toggle a
       line of its own and doubled the page. The description wraps instead, and the
       control stays at the top, level with the label it belongs to. -->
  <component :is="tag" class="flex min-h-13 items-start justify-between gap-3 py-3.5 text-sm font-medium">
    <span class="min-w-0 flex-1">
      <span class="inline-flex items-center gap-2">
        {{ label }}
        <slot name="label-suffix" />
      </span>
      <span v-if="description" class="text-base-content/60 mt-0.5 block text-xs font-normal">
        {{ description }}
      </span>
    </span>
    <span class="flex shrink-0 items-center gap-2">
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
