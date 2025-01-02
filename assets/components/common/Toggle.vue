<template>
  <LabeledInput>
    <template #label>
      <slot />
    </template>
    <template #input>
      <input type="checkbox" class="toggle toggle-primary" v-model="modelValue" v-bind="$attrs" />
    </template>
  </LabeledInput>
</template>

<script lang="ts" setup>
const modelValue = defineModel<boolean>();
</script>
