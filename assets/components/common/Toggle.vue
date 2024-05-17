<template>
  <LabeledInput>
    <template #label>
      <slot />
    </template>
    <template #input>
      <input type="checkbox" class="toggle toggle-primary" v-model="modelValue" />
    </template>
  </LabeledInput>
</template>

<script lang="ts" setup>
const { modelValue } = defineModels<{
  modelValue: boolean;
}>();
</script>
