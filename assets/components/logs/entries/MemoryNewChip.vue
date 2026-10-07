<template>
  <!-- The NEW mark itself, shared by the chip on the line and the log details
       drawer so the two always look the same. -->
  <span class="memory-new-chip">{{ $t("label.memory-new") }}</span>
</template>

<style scoped>
@reference "@/main.css";
.memory-new-chip {
  color: var(--color-warning);
  border: 1px solid color-mix(in oklab, var(--color-warning) 45%, transparent);
  @apply inline-flex shrink-0 items-center rounded-xs px-1.5 py-px font-sans text-[0.62rem] font-bold tracking-wider uppercase;
}
</style>
