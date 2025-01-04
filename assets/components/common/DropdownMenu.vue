<template>
  <details class="dropdown" ref="details" v-on-click-outside="close">
    <summary class="btn btn-primary flex-nowrap" v-bind="$attrs">
      <slot name="trigger"> {{ label }} <carbon:caret-down /></slot>
    </summary>
    <ul class="menu dropdown-content rounded-box border-base-content/20 bg-base z-50 mt-1 w-52 border p-2 shadow-sm">
      <slot>
        <li v-for="item in options">
          <a @click="update(item.value as T)">
            <mdi:check class="w-4" v-if="modelValue == item.value" />
            <div v-else class="w-4"></div>
            {{ item.label }}
          </a>
        </li>
      </slot>
    </ul>
  </details>
</template>

<script lang="ts" setup generic="T">
import { vOnClickOutside } from "@vueuse/components";
type DropdownItem = {
  label: string;
  value: T;
};

const model = defineModel<T>();

const { options, defaultLabel = "" } = defineProps<{
  options: DropdownItem[];
  defaultLabel?: string;
}>();

const label = computed(() => options.find((item) => item.value === model.value)?.label ?? defaultLabel);
const details = ref<HTMLElement | null>(null);
const close = () => details.value?.removeAttribute("open");

const update = (value: T) => {
  model.value = value;
  close();
};
</script>
