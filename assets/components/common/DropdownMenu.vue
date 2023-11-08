<template>
  <details class="dropdown" ref="details" v-on-click-outside="close">
    <summary class="btn btn-primary flex-nowrap" v-bind="$attrs">
      <slot name="trigger"> {{ values[modelValue] ?? defaultLabel }} <carbon:caret-down /></slot>
    </summary>
    <ul class="menu dropdown-content rounded-box z-50 mt-1 w-52 border border-base-content/20 bg-base p-2 shadow">
      <slot>
        <li v-for="item in options">
          <a @click="modelValue = item.value">
            <mdi:check class="w-4" v-if="modelValue == item.value" />
            <div v-else class="w-4"></div>
            {{ item.label }}
          </a>
        </li>
      </slot>
    </ul>
  </details>
</template>

<script lang="ts" setup>
import { vOnClickOutside } from "@vueuse/components";
type DropdownItem = {
  label: string;
  value: string;
};
const { options = [], defaultLabel = "" } = defineProps<{ options?: DropdownItem[]; defaultLabel?: string }>();
const { modelValue } = defineModels<{
  modelValue: string;
}>();

const values = computed(() =>
  options.reduce(
    (acc, curr) => {
      acc[curr.value] = curr.label;
      return acc;
    },
    {} as Record<string, string>,
  ),
);

const details = ref<HTMLElement | null>(null);
const close = () => details.value?.removeAttribute("open");
watch(modelValue, () => close());
</script>
