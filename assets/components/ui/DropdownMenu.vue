<template>
  <Popover
    placement="bottom-end"
    panel-class="rounded-box border-base-content/20 bg-base-200 max-h-72 w-48 overflow-auto border p-2 shadow-sm"
  >
    <template #trigger>
      <!-- `plain` is for a picker that sits beside the surface's real action and must
           not compete with it for the one primary slot. -->
      <button type="button" class="btn flex-nowrap" :class="{ 'btn-primary': !plain }" v-bind="$attrs">
        <slot name="trigger"> {{ label }} <carbon:caret-down /></slot>
      </button>
    </template>
    <ul class="menu w-full flex-nowrap p-0">
      <slot>
        <li v-for="item in options">
          <a @click="model = item.value as T">
            <mdi:check class="w-4" v-if="modelValue == item.value" />
            <div v-else class="w-4"></div>
            {{ item.label }}
          </a>
        </li>
      </slot>
    </ul>
  </Popover>
</template>

<script lang="ts" setup generic="T">
defineOptions({ inheritAttrs: false });

type DropdownItem = {
  label: string;
  value: T;
};

const model = defineModel<T>();

const {
  options,
  defaultLabel = "",
  plain = false,
} = defineProps<{
  options: DropdownItem[];
  defaultLabel?: string;
  plain?: boolean;
}>();

const label = computed(() => options.find((item) => item.value === model.value)?.label ?? defaultLabel);
</script>
