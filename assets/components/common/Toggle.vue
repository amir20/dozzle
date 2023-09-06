<template>
  <label class="label cursor-pointer justify-start gap-4 font-normal">
    <input type="checkbox" class="toggle toggle-primary" :checked="modelValue" @click="toggle()" />
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
