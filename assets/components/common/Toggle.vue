<template>
  <label class="label inline-flex cursor-pointer gap-4 font-normal">
    <input
      type="checkbox"
      class="toggle bg-primary hover:bg-primary-focus"
      style="--tglbg: var(--scheme-main-ter-color)"
      :checked="modelValue"
      @click="toggle()"
    />
    <slot />
  </label>
</template>

<script lang="ts" setup>
const { modelValue } = defineModels<{
  modelValue: boolean;
}>();

const toggle = useToggle(modelValue);
</script>
