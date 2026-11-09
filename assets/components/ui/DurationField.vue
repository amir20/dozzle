<template>
  <fieldset class="fieldset">
    <legend class="fieldset-legend" :class="stops ? 'text-sm' : 'text-lg'">{{ label }}</legend>
    <div class="flex items-center gap-3">
      <!-- The hint below already states the value, so a snapped slider has no readout of its own. -->
      <input
        v-if="stops"
        v-model.number="stopIndex"
        type="range"
        min="0"
        :max="stops.length - 1"
        step="1"
        class="range range-primary range-xs"
        :aria-label="label"
        :aria-valuetext="formatDuration(model, locale || undefined)"
      />
      <template v-else>
        <input v-model.number="slider" type="range" :min="min" :max="max" :step="step" class="range range-primary" />
        <label class="input input-sm w-32 shrink-0">
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
      </template>
    </div>
    <p class="text-base-content/60 mt-1 text-xs">
      <template v-if="model === 0 && zeroHint">{{ zeroHint }}</template>
      <template v-else>{{ hint(formatDuration(model, locale || undefined)) }}</template>
    </p>
  </fieldset>
</template>

<script lang="ts" setup>
const {
  min = 0,
  max = 0,
  step = 1,
  stops,
} = defineProps<{
  label: string;
  /** Renders the description for the current value, e.g. "5m between alerts". */
  hint: (duration: string) => string;
  /** Shown instead of `hint` when the value is 0, for fields where 0 means "off". */
  zeroHint?: string;
  min?: number;
  max?: number;
  step?: number;
  /** Fixed values in seconds the slider snaps between, for ranges too wide for a linear slider. */
  stops?: number[];
}>();

const model = defineModel<number>({ required: true });

// The slider snaps to `step`; the number input lets you land on an exact value the
// slider can't reach. Both write through to the same model.
const slider = computed({
  get: () => model.value,
  set: (v) => (model.value = clamp(v)),
});

// A saved value between two stops shows at the nearest one and stays as it is until moved.
const stopIndex = computed({
  get: () => {
    const distances = stops!.map((stop) => Math.abs(stop - model.value));
    return distances.indexOf(Math.min(...distances));
  },
  set: (index) => (model.value = stops![index]),
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
