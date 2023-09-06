<template>
  <details class="dropdown" ref="details">
    <summary class="btn border-none bg-primary hover:bg-primary-focus">
      {{ values[modelValue] }} <carbon:caret-down />
    </summary>
    <ul class="menu dropdown-content rounded-box z-[1] w-52 p-2 shadow" v-on-click-outside="close">
      <li v-for="item in options">
        <a @click="modelValue = item.value"> {{ item.label }} </a>
      </li>
    </ul>
  </details>
</template>

<script lang="ts" setup>
import { vOnClickOutside } from "@vueuse/components";
type DropdownItem = {
  label: string;
  value: string;
};
const { options } = defineProps<{ options: DropdownItem[] }>();
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

<style scoped lang="scss"></style>
