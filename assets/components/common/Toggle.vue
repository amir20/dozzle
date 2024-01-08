<template>
  <labeled-input>
    <template #label>
      <slot />
    </template>
    <template #input>
      <input type="checkbox" class="toggle toggle-primary" v-model="modelValue" />
    </template>
  </labeled-input>
</template>

<script lang="ts" setup>
const { modelValue } = defineModels<{
  modelValue: boolean;
}>();
</script>
