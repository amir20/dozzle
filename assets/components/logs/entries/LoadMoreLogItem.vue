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
  const list = root.value?.closest("ul");
  const rowsBefore = list?.childElementCount;
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
  // With folding on, older repeats of the top line join its run, which redraws
  // that row as a new element; a run shows its newest line, so the row keeps its
  // id. A replaced window took the anchor away for good, and its own scroll
  // position stands. The last page swaps this row for the start edge but keeps
  // the anchor, so that page still holds the reader's place.
  const current = anchor?.isConnected ? anchor : anchor?.id ? document.getElementById(anchor.id) : null;
  if (logEntry.rememberScrollPosition && current && anchorTop !== undefined) {
    scroller.scrollTop += current.getBoundingClientRect().top - anchorTop;
  }
  // With folding on, a page of older repeats only grows the top run's ×N. No row
  // was added, so the next page would be the same, all the way back to the start.
  if (more && list?.childElementCount !== rowsBefore && inView(scroller)) await load();
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
