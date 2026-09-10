<template>
  <fieldset class="fieldset">
    <legend class="fieldset-legend text-lg">{{ label }}</legend>
    <div class="flex items-center gap-3">
      <input v-model.number="slider" type="range" :min="min" :max="max" :step="step" class="range range-primary" />
      <label class="input input-sm w-28 shrink-0">
        <input
          v-model.number="typed"
          type="number"
          :min="min"
          :max="max"
          :step="step"
          class="text-right"
          :aria-label="label"
          @blur="commitTyped"
          @keydown.enter.prevent="commitTyped"
        />
        <span class="label">{{ $t("notifications.alert-form.seconds-unit") }}</span>
      </label>
    </div>
    <p class="text-base-content/60 mt-1 text-xs">
      <template v-if="model === 0 && zeroHint">{{ zeroHint }}</template>
      <template v-else>{{ hint(formatDuration(model, locale || undefined)) }}</template>
    </p>
  </fieldset>
</template>

<script lang="ts" setup>
const { min, max, step } = defineProps<{
  label: string;
  /** Renders the description for the current value, e.g. "5m between alerts". */
  hint: (duration: string) => string;
  /** Shown instead of `hint` when the value is 0, for fields where 0 means "off". */
  zeroHint?: string;
  min: number;
  max: number;
  step: number;
}>();

const model = defineModel<number>({ required: true });

// The slider snaps to `step`; the number input lets you land on an exact value the
// slider can't reach. Both write through to the same model.
const slider = computed({
  get: () => model.value,
  set: (v) => (model.value = clamp(v)),
});

const typed = ref(model.value);
watch(model, (v) => (typed.value = v));

function clamp(value: number) {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, Math.round(value)));
}

function commitTyped() {
  const next = clamp(typed.value);
  typed.value = next;
  model.value = next;
}
</script>
