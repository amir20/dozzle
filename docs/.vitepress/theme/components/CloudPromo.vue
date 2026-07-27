<script lang="ts" setup>
import { withBase } from "vitepress";
</script>

<template>
  <section class="mx-auto max-w-[1152px] px-6 pt-4 pb-8">
    <div
      class="flex flex-col items-start gap-6 rounded-xl border border-solid border-(--vp-c-divider) bg-(--vp-c-bg-soft) p-8 md:flex-row md:items-center md:justify-between"
    >
      <div class="max-w-2xl">
        <div class="mb-2 flex items-center gap-3">
          <h2 class="m-0! border-0! p-0! text-xl font-semibold tracking-tight">Dozzle Cloud</h2>
          <span class="rounded-full bg-(--vp-c-brand-dimm) px-2.5 py-0.5 text-xs font-medium text-(--vp-c-brand)">
            Optional
          </span>
        </div>
        <p class="m-0! text-(--vp-c-text-2)">
          A managed companion that links your self-hosted instances together, summarizes container events, fans alerts
          out across channels, and answers questions about your infrastructure from chat. Dozzle itself stays fully open
          source.
        </p>
      </div>
      <div class="flex shrink-0 flex-wrap gap-3">
        <a
          class="rounded-3xl border border-solid border-(--vp-button-brand-border) bg-(--vp-button-brand-bg) px-5 py-2 text-sm leading-6 font-semibold text-(--vp-button-brand-text) no-underline! transition-colors hover:bg-(--vp-button-brand-hover-bg)"
          href="https://cloud.dozzle.dev"
          >Try Cloud</a
        >
        <a
          class="rounded-3xl border border-solid border-(--vp-button-alt-border) bg-(--vp-button-alt-bg) px-5 py-2 text-sm leading-6 font-semibold text-(--vp-button-alt-text) no-underline! transition-colors hover:bg-(--vp-button-alt-hover-bg)"
          :href="withBase('/guide/dozzle-cloud')"
          >Learn More</a
        >
      </div>
    </div>
  </section>
</template>
