<template>
  <fieldset class="fieldset">
    <legend class="fieldset-legend text-sm">{{ label }}</legend>
    <input
      v-model.number="stopIndex"
      type="range"
      min="0"
      :max="allStops.length - 1"
      step="1"
      class="range range-primary range-xs"
      :aria-label="label"
      :aria-valuetext="formatDuration(model, locale || undefined)"
    />
    <p class="text-base-content/60 mt-1 text-xs">
      <template v-if="model === 0 && zeroHint">{{ zeroHint }}</template>
      <template v-else>{{ hint(formatDuration(model, locale || undefined)) }}</template>
    </p>
  </fieldset>
</template>

<script lang="ts" setup>
const { stops } = defineProps<{
  label: string;
  /** Renders the description for the current value, e.g. "5m between alerts". */
  hint: (duration: string) => string;
  /** Shown instead of `hint` when the value is 0, for fields where 0 means "off". */
  zeroHint?: string;
  /** Values in seconds the slider snaps between. */
  stops: number[];
}>();

const model = defineModel<number>({ required: true });

// A saved value between two stops gets a stop of its own, so the slider and the hint agree.
// Fixed at mount: if it left the list when moved off, the stops would shift under the drag.
const allStops = stops.includes(model.value) ? stops : [...stops, model.value].sort((a, b) => a - b);

const stopIndex = computed({
  get: () => allStops.indexOf(model.value),
  set: (index) => (model.value = allStops[index]),
});
</script>
