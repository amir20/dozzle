<template>
  <label class="flex items-start gap-2 text-sm">
    <input v-model="checked" type="checkbox" class="checkbox checkbox-sm mt-0.5" />
    <span class="flex flex-col">
      <span>{{ $t("updates.watch-in-cloud") }}</span>
      <span class="text-base-content/60 text-xs">{{ $t("updates.watch-in-cloud-hint") }}</span>
    </span>
  </label>
</template>

<script lang="ts" setup>
// "Have Dozzle Cloud watch this update", shared by the single update dialog
// and the bulk drawer so both ask it the same way.
const checked = defineModel<boolean>({ required: true });
</script>
