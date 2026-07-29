<script lang="ts" setup>
import { withBase } from "vitepress";
import InstallCommand from "./InstallCommand.vue";
</script>

<template>
  <section class="mx-auto mt-20 max-w-[1152px] px-6">
    <div
      class="flex flex-col items-center rounded-2xl border border-solid border-(--vp-c-divider) bg-(--vp-c-bg-alt) px-6 py-16"
    >
      <h2 class="m-0! text-center text-2xl font-semibold text-(--vp-c-text-1) md:text-3xl">
        Start watching your containers
      </h2>
      <p class="mt-3! mb-0! max-w-xl text-center text-(--vp-c-text-2)">
        One command, no agent to install, no account to create.
      </p>

      <div class="mt-8 w-full">
        <InstallCommand :hint="false" dense />
      </div>

      <div class="mt-10 flex flex-wrap items-center justify-center gap-3">
        <a class="cta-button cta-brand" :href="withBase('/guide/getting-started')">Get Started</a>
        <a class="cta-button cta-alt" href="https://github.com/amir20/dozzle">
          <Icon icon="simple-icons:github" class="size-4" />
          View on GitHub
        </a>
      </div>

      <p class="mt-8! mb-0! text-sm text-(--vp-c-text-2)">MIT licensed and free forever.</p>
    </div>
  </section>
</template>

<style scoped>
/* Mirror VitePress's own button tokens so these match the hero buttons in both themes. */
.cta-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.5rem;
  border: 1px solid transparent;
  border-radius: 9999px;
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.5;
  text-decoration: none !important;
  transition:
    background-color 0.25s,
    border-color 0.25s,
    color 0.25s;
}

.cta-brand {
  border-color: var(--vp-button-brand-border);
  background-color: var(--vp-button-brand-bg);
  color: var(--vp-button-brand-text);
}

.cta-brand:hover {
  border-color: var(--vp-button-brand-hover-border);
  background-color: var(--vp-button-brand-hover-bg);
  color: var(--vp-button-brand-hover-text);
}

.cta-alt {
  border-color: var(--vp-button-alt-border);
  background-color: var(--vp-button-alt-bg);
  color: var(--vp-button-alt-text);
}

.cta-alt:hover {
  border-color: var(--vp-button-alt-hover-border);
  background-color: var(--vp-button-alt-hover-bg);
  color: var(--vp-button-alt-hover-text);
}
</style>
