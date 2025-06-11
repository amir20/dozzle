<template>
  <div ref="root" class="flex min-h-[1px] flex-1 content-center justify-center">
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

useIntersectionObserver(root, async (entries) => {
  if (entries[0].intersectionRatio <= 0) return;
  if (isLoading.value) return;
  const scrollingParent = root.value?.closest("[data-scrolling]") || document.documentElement;
  const previousHeight = scrollingParent.scrollHeight;
  isLoading.value = true;
  await logEntry.loadMore();
  isLoading.value = false;
  await nextTick();
  if (logEntry.rememberScrollPosition) {
    scrollingParent.scrollTop += scrollingParent.scrollHeight - previousHeight;
  }
});
</script>

<style scoped></style>
