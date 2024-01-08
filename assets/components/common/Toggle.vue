<template>
  <label class="label cursor-pointer gap-4">
    <div class="label-text"><slot /></div>
    <input type="checkbox" class="toggle toggle-primary" v-model="modelValue" />
  </label>
</template>

<script lang="ts" setup>
const { modelValue } = defineModels<{
  modelValue: boolean;
}>();
</script>
