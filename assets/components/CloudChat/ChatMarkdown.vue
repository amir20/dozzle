<template>
  <div class="chat-md" v-html="html"></div>
</template>

<script lang="ts" setup>
import MarkdownIt from "markdown-it";
// markdown-it v15 ships its own types and no longer has deep /lib entry points;
// `MarkdownIt` is both the default value and a named type, hence the alias.
import type { Env, MarkdownIt as MarkdownItInstance, MarkdownItOptions, Renderer, Token } from "markdown-it";

const { text } = defineProps<{ text: string }>();

// The assistant answers in markdown, so a bare `whitespace-pre-wrap` shows the
// reader `**doligence-api-1**` and a stray fence. Same renderer and same
// options as the cloud console's chat, so an answer reads the same in both.
//
// `html: false` is what makes rendering model output safe: source HTML is
// escaped rather than parsed, so nothing in a quoted log line becomes markup.
const md: MarkdownItInstance = new MarkdownIt({ html: false, linkify: true, breaks: true });

const defaultLinkRender =
  md.renderer.rules.link_open ||
  ((tokens: Token[], idx: number, options: Required<MarkdownItOptions>, _env: Env | undefined, self: Renderer) =>
    self.renderToken(tokens, idx, options));

md.renderer.rules.link_open = (
  tokens: Token[],
  idx: number,
  options: Required<MarkdownItOptions>,
  env: Env | undefined,
  self: Renderer,
) => {
  tokens[idx]!.attrSet("target", "_blank");
  tokens[idx]!.attrSet("rel", "noopener noreferrer");
  return defaultLinkRender(tokens, idx, options, env, self);
};

const html = computed(() => md.render(text));
</script>

<style scoped>
@reference "@/main.css";

/* Unscoped selectors under one class, because v-html output carries no scope
   attribute. */
.chat-md :deep(p) {
  @apply mb-2 last:mb-0;
}

.chat-md :deep(ul),
.chat-md :deep(ol) {
  @apply mb-2 list-outside pl-5 last:mb-0;
}

.chat-md :deep(ul) {
  @apply list-disc;
}

.chat-md :deep(ol) {
  @apply list-decimal;
}

.chat-md :deep(li) {
  @apply mb-0.5;
}

/* The assistant quotes log lines and container names back at you; they only
   read as log content in the face the log surfaces use. */
.chat-md :deep(code) {
  @apply bg-base-content/10 rounded px-1 font-mono text-[0.85em];
}

.chat-md :deep(pre) {
  @apply bg-base-content/5 border-base-content/10 my-2 overflow-x-auto rounded-md border p-2;
}

.chat-md :deep(pre code) {
  @apply bg-transparent p-0;
}

.chat-md :deep(a) {
  @apply link link-primary;
}

.chat-md :deep(h1),
.chat-md :deep(h2),
.chat-md :deep(h3),
.chat-md :deep(h4) {
  @apply mt-3 mb-1 font-semibold first:mt-0;
}

.chat-md :deep(hr) {
  @apply border-base-content/10 my-3;
}

.chat-md :deep(table) {
  @apply my-2 block overflow-x-auto;
}

.chat-md :deep(th),
.chat-md :deep(td) {
  @apply border-base-content/10 border px-2 py-1 text-left;
}

.chat-md :deep(blockquote) {
  @apply border-base-content/20 text-base-content/60 my-2 border-l-2 pl-3;
}
</style>
