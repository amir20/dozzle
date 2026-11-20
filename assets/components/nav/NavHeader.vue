<template>
  <div class="mb-2 flex h-7 shrink-0 items-center gap-1">
    <button type="button" class="icon-btn nav-btn" v-if="back" :title="back" :aria-label="back" @click="$emit('back')">
      <mdi:chevron-left class="size-4" />
    </button>

    <!-- The filter takes over the title's slot rather than adding a row, so a short
         list never pays for it in height. -->
    <label
      v-if="filtering"
      class="border-base-content/15 bg-base-content/5 focus-within:border-base-content/30 flex h-7 min-w-0 flex-1 items-center gap-1.5 rounded-md border px-1.5 transition-colors"
    >
      <mdi:magnify class="text-base-content/50 size-4 shrink-0" />
      <input
        ref="input"
        v-model="filter"
        type="text"
        class="min-w-0 flex-1 bg-transparent text-sm outline-none"
        :placeholder="$t('placeholder.filter')"
        :aria-label="$t('tooltip.filter-list')"
        @keydown.esc.stop.prevent="close()"
        @blur="!filter && close()"
      />
      <button
        type="button"
        class="text-base-content/50 hover:text-base-content shrink-0 transition-colors"
        :title="$t('toolbar.clear')"
        :aria-label="$t('toolbar.clear')"
        @mousedown.prevent
        @click="close()"
      >
        <mdi:close class="size-3.5" />
      </button>
    </label>

    <div class="flex min-w-0 flex-1 items-center gap-1.5" v-else>
      <slot name="title">
        <span class="truncate text-[0.9375rem] font-medium">{{ title }}</span>
      </slot>
    </div>

    <div class="flex shrink-0 items-center gap-0.5">
      <button
        type="button"
        class="icon-btn nav-btn"
        v-if="filterable && !filtering"
        :title="$t('tooltip.filter-list')"
        :aria-label="$t('tooltip.filter-list')"
        @click="open()"
      >
        <mdi:magnify class="size-4" />
      </button>
      <slot name="actions" />
    </div>
  </div>
</template>

<script lang="ts" setup>
const {
  title,
  back,
  filterable = false,
} = defineProps<{
  title?: string;
  /** Label for the back affordance. Omitted means there is nowhere to go back to. */
  back?: string;
  /** Shows the magnifier that swaps the title for a filter input bound to `v-model:filter`. */
  filterable?: boolean;
}>();

defineEmits<{ back: [] }>();

const filter = defineModel<string>("filter", { default: "" });

const filtering = ref(false);
const input = useTemplateRef<HTMLInputElement>("input");

const open = async () => {
  filtering.value = true;
  await nextTick();
  input.value?.focus();
};

const close = () => {
  filter.value = "";
  filtering.value = false;
};

// A panel that stops offering the filter (the host menu sliding back to its host
// list) must not leave a hidden query narrowing what comes back.
watch(
  () => filterable,
  (on) => !on && close(),
);
</script>
