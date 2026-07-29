<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { withBase } from "vitepress";

const { hint = true, dense = false } = defineProps<{
  hint?: boolean;
  dense?: boolean;
}>();

const command =
  "docker run -d -v /var/run/docker.sock:/var/run/docker.sock -v dozzle_data:/data -p 8080:8080 amir20/dozzle:latest";

const { copy, copied } = useClipboard({ source: command, copiedDuring: 2000 });
</script>

<template>
  <div class="mx-auto flex max-w-[1152px] flex-col items-center gap-3 px-6" :class="dense ? 'mb-0' : 'mb-14'">
    <div
      class="flex w-full max-w-4xl items-start gap-3 rounded-lg border border-solid border-(--vp-c-divider) bg-(--vp-c-bg-alt) py-3 pr-3 pl-4"
    >
      <span class="shrink-0 font-mono text-xs leading-6 text-(--vp-c-text-3) select-none md:text-sm">$</span>
      <code
        class="min-w-0 flex-1 bg-transparent! p-0! font-mono text-xs leading-6 break-words whitespace-pre-wrap text-(--vp-c-text-1) md:text-sm"
        >{{ command }}</code
      >
      <button
        type="button"
        class="shrink-0 cursor-pointer rounded-md border border-solid border-(--vp-c-divider) bg-(--vp-c-bg) px-3 py-1.5 text-xs font-medium text-(--vp-c-text-2) transition-colors hover:border-(--vp-c-brand) hover:text-(--vp-c-brand)"
        :aria-label="copied ? 'Command copied' : 'Copy command to clipboard'"
        @click="copy()"
      >
        {{ copied ? "Copied" : "Copy" }}
      </button>
    </div>
    <p v-if="hint" class="m-0! text-sm text-(--vp-c-text-2)">
      Then open <span class="font-mono">http://localhost:8080</span>. Prefer Compose, Swarm, or Kubernetes? See the
      <a :href="withBase('/guide/getting-started')">install guide</a>.
    </p>
  </div>
</template>
