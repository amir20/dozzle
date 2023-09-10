<template>
  <label class="label inline-flex cursor-pointer gap-4 font-normal">
    <input
      type="checkbox"
      class="toggle bg-primary"
      style="--tglbg: var(--scheme-main-ter)"
      :checked="modelValue"
      @click="toggle()"
    />
    <span>
      <slot />
    </span>
  </label>
</template>

<script lang="ts" setup>
const { modelValue } = defineModels<{
  modelValue: boolean;
}>();

const toggle = useToggle(modelValue);
</script>
