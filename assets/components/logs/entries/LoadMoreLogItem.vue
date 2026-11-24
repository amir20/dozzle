<template>
  <div ref="root" class="flex min-h-px flex-1 content-center justify-center">
    <span class="loading loading-bars loading-md text-primary m-2" v-show="isLoading"></span>
  </div>
</template>
<script lang="ts" setup>
import { LoadMoreLogEntry } from "@/models/LogEntry";

const { logEntry } = defineProps<{
  logEntry: LoadMoreLogEntry;
}>();

const isLoading = ref(false);
const root = ref<HTMLElement>();

// The observer only reports changes, so a load that leaves the row on screen has
// to look for itself.
function inView(scroller: Element) {
  if (!root.value?.isConnected) return false;
  const row = root.value.getBoundingClientRect();
  const box =
    scroller === document.documentElement ? { top: 0, bottom: window.innerHeight } : scroller.getBoundingClientRect();
  return row.bottom > box.top && row.top < box.bottom;
}

async function load() {
  const scroller = root.value?.closest("[data-scrolling]") || document.documentElement;
  // The row under the loader is what the reader is looking at. Older lines can
  // land above it, or merge in by time below the view, and the stream keeps
  // appending at the bottom; only what lands above it should move the view.
  const anchor = root.value?.closest("li")?.nextElementSibling;
  const anchorTop = anchor?.getBoundingClientRect().top;
  isLoading.value = true;
  let more = false;
  try {
    more = await logEntry.loadMore();
  } catch (err) {
    console.error(err);
  } finally {
    isLoading.value = false;
  }
  await nextTick();
  // A replaced window took this row away, and its own scroll position stands.
  if (!root.value?.isConnected) return;
  if (logEntry.rememberScrollPosition && anchor?.isConnected && anchorTop !== undefined) {
    scroller.scrollTop += anchor.getBoundingClientRect().top - anchorTop;
  }
  if (more && inView(scroller)) await load();
}

// Covers the loads that follow one another too, not just the one in flight.
let busy = false;
useIntersectionObserver(root, async (entries) => {
  if (entries[0].intersectionRatio <= 0) return;
  if (busy) return;
  busy = true;
  try {
    await load();
  } finally {
    busy = false;
  }
});
</script>

<style scoped></style>
