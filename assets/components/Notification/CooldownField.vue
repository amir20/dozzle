<template>
  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">{{ $t("notifications.alert-form.cooldown-label") }}</legend>
    <input v-model.number="model" type="range" min="0" max="3600" step="10" class="range range-primary" />
    <p class="text-base-content/50 mt-1 text-xs">
      <template v-if="model === 0">{{ $t("notifications.alert-form.no-cooldown") }}</template>
      <template v-else>{{
        $t("notifications.alert-form.cooldown-hint", { duration: formatDuration(model, locale || undefined) })
      }}</template>
    </p>
  </fieldset>
</template>

<script lang="ts" setup>
const model = defineModel<number>({ required: true });
</script>
