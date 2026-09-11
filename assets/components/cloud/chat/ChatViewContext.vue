<template>
  <!--
    What gets sent with the question.

    If the assistant silently knows what is on your screen you cannot tell what
    it knows, and one wrong assumption makes you distrust all of it. Above the
    composer this is the evidence you are about to hand over, so it is stated at
    reading size; inside the thread it is a record of what a turn was asked
    with, so it shrinks to a footnote.
  -->
  <div v-if="compact" class="text-base-content/40 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs">
    <mdi:eye-outline class="size-3.5 shrink-0" />
    <template v-for="(part, i) in parts" :key="part.key">
      <span v-if="i" aria-hidden="true">·</span>
      <span :class="part.name ? 'text-base-content/70 truncate font-mono' : ''">{{ part.label }}</span>
    </template>
  </div>

  <div v-else class="border-base-content/15 bg-base-200/40 rounded-lg border px-3 py-2">
    <div class="text-base-content/40 mb-1 text-[11px] font-semibold tracking-wide uppercase">
      {{ $t("cloud-chat.evidence") }}
    </div>
    <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
      <template v-for="(part, i) in parts" :key="part.key">
        <span v-if="i" class="text-base-content/25" aria-hidden="true">·</span>
        <span :class="part.name ? 'truncate font-mono font-semibold' : 'text-base-content/60'">{{ part.label }}</span>
      </template>
    </div>

    <!-- A line the reader pointed at is evidence as much as the window is, so
         it belongs in this card rather than in a second one under it. -->
    <template v-if="hasSlotContent()">
      <div class="bg-base-content/10 my-2 h-px"></div>
      <slot />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { Comment, Fragment, Text, type VNode } from "vue";
import type { ViewContext } from "@/composable/logs/viewContext";

const { view, compact = false } = defineProps<{ view: ViewContext; compact?: boolean }>();
const { t } = useI18n();
const slots = useSlots();

// `$slots.default` is truthy the moment a caller writes slot content at all,
// even when that content is a single `v-if` that rendered nothing. The composer
// always passes the focused line, so testing it drew a divider with nothing
// under it whenever no line was focused. Called from the template so it is
// re-evaluated, and re-tracked, on every render.
function hasSlotContent() {
  return (slots.default?.() ?? []).some(isRendered);
}

function isRendered(node: VNode): boolean {
  if (node.type === Comment) return false;
  if (node.type === Fragment) return (node.children as VNode[] | null)?.some(isRendered) ?? false;
  if (node.type === Text) return !!String(node.children ?? "").trim();
  return true;
}

const parts = computed(() => {
  const out: { key: string; label: string; name?: boolean }[] = [];
  const names = view.containers.map((c) => c.name);

  if (names.length === 1) {
    out.push({ key: "container", label: names[0], name: true });
  } else if (names.length > 1) {
    out.push({ key: "container", label: t("cloud-chat.n-containers", { n: names.length }) });
  } else if (view.target) {
    out.push({ key: "target", label: view.target, name: true });
  }

  if (view.lines?.length) out.push({ key: "lines", label: t("cloud-chat.lines-in-view", { n: view.lines.length }) });
  if (view.search) out.push({ key: "search", label: `"${view.search}"` });
  if (view.levels?.length) out.push({ key: "levels", label: view.levels.join(" / ") });
  if (view.historical) out.push({ key: "historical", label: t("cloud-chat.historical") });

  return out;
});
</script>
