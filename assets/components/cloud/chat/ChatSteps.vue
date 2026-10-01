<template>
  <!-- Live: what is happening now, with the lookups already done listed quietly
       under it, so a tool call that takes 200ms stays on screen instead of
       flashing past. -->
  <div v-if="live" class="space-y-2">
    <div class="flex items-center gap-2 text-sm">
      <span class="loading loading-dots loading-xs text-info"></span>
      <span class="text-base-content/70 min-w-0 flex-1 truncate">{{ current ? stepText(current) : fallback }}</span>
      <span v-if="elapsed >= 3" class="text-base-content/40 shrink-0 font-mono text-xs tabular-nums"
        >{{ elapsed }}s</span
      >
    </div>
    <ul v-if="visible.length" class="border-base-content/10 ml-1 space-y-0.5 border-l pl-3">
      <li v-if="hidden" class="text-base-content/40 text-xs">{{ $t("cloud-chat.earlier", { n: hidden }) }}</li>
      <li v-for="step in visible" :key="step.id">
        <ChatStepRow :step="step" />
      </li>
    </ul>
  </div>

  <!-- Folded above a finished answer. Nothing to fold when no tool ran. -->
  <div v-else-if="lookups.length">
    <button
      type="button"
      class="text-base-content/50 hover:text-base-content/80 flex items-center gap-1 rounded text-xs transition-colors focus:outline-none focus-visible:ring-2"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <mdi:chevron-right class="size-3.5 transition-transform" :class="expanded ? 'rotate-90' : ''" />
      {{ $t("cloud-chat.worked-for", { time: seconds(ms ?? 0) }) }}
    </button>
    <ul v-if="expanded" class="border-base-content/10 mt-1 ml-1.5 space-y-0.5 border-l pl-3">
      <li v-for="step in lookups" :key="step.id">
        <ChatStepRow :step="step" />
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import type { ChatStep } from "@/composable/cloud/cloudChat";
import { stepText } from "./chatSteps";

// A turn's progress trail. Only tool calls are listed: a model round ("Reading
// the question") is worth showing while it runs, as the current line, and says
// nothing once the lookups it chose are listed under it.
const props = defineProps<{
  steps: ChatStep[];
  live?: boolean;
  /** When the turn began, for the live timer. */
  startedAt?: number;
  /** Total turn time, for the folded line. */
  ms?: number;
  /** The current line before the first step arrives. */
  fallback?: string;
}>();

const current = computed(() => props.steps.findLast((s) => s.state === "running"));
const lookups = computed(() => props.steps.filter((s) => s.tool && s !== current.value));

// Live, older lookups fold into a count so the line stays a few rows tall
// however many rounds the turn takes.
const LIVE_VISIBLE = 3;
const hidden = computed(() => (props.live ? Math.max(0, lookups.value.length - LIVE_VISIBLE) : 0));
const visible = computed(() => lookups.value.slice(hidden.value));

const expanded = ref(false);

// The clock only ticks while live; a finished answer keeps no timer.
const now = ref(Date.now());
const { pause, resume } = useIntervalFn(() => (now.value = Date.now()), 1000, { immediate: false });
watch(
  () => props.live,
  (live) => (live ? resume() : pause()),
  { immediate: true },
);
const elapsed = computed(() =>
  props.live && props.startedAt ? Math.max(0, Math.floor((now.value - props.startedAt) / 1000)) : 0,
);

function seconds(ms: number) {
  const s = ms / 1000;
  return s < 10 ? `${s.toFixed(1)}s` : `${Math.round(s)}s`;
}
</script>
