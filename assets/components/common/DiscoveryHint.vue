<template>
  <div class="dropdown" :class="align === 'end' ? 'dropdown-end' : ''">
    <button
      tabindex="0"
      role="button"
      class="text-base-content/25 hover:text-primary shrink-0 cursor-pointer transition-colors"
      :title="title"
      :aria-label="title"
    >
      <slot name="icon"><mdi:lightbulb-on-outline class="size-4" /></slot>
    </button>
    <div
      tabindex="0"
      class="dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 mt-1 border p-3 text-xs shadow-sm"
      :style="{ width: `${width}px` }"
    >
      <div class="text-base-content/60 mb-2 text-[11px] tracking-wide uppercase">{{ title }}</div>
      <slot></slot>
      <div class="mt-2 flex items-center justify-between gap-2">
        <div><slot name="action"></slot></div>
        <div class="flex items-center gap-2">
          <a
            v-if="docs"
            :href="docs"
            target="_blank"
            rel="noopener noreferrer"
            class="link link-hover text-base-content/60"
            >{{ $t("hint.learn-more") }}</a
          >
          <button type="button" class="link link-hover text-base-content/60" @click="$emit('dismiss')">
            {{ $t("hint.dismiss") }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
/**
 * A dismissible "did you know" popover hung off a small icon. Use it to teach an opt-in
 * feature at the point where its absence is visible, not as a general tooltip.
 */
const {
  title,
  docs = undefined,
  width = 320,
  align = "start",
} = defineProps<{
  title: string;
  /** Documentation URL for the "Learn more" link. Omitted, the link is hidden. */
  docs?: string;
  width?: number;
  align?: "start" | "end";
}>();

defineEmits<{ dismiss: [] }>();
</script>
