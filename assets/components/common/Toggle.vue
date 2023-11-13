<template>
  <label class="label inline-flex cursor-pointer gap-4 font-normal">
    <input type="checkbox" class="toggle toggle-primary" v-model="modelValue" />
    <slot />
  </label>
</template>

<script lang="ts" setup>
const { modelValue } = defineModels<{
  modelValue: boolean;
}>();
</script>
