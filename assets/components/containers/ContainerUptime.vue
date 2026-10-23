<template>
  <time v-if="status" :datetime="status.since.toISOString()">{{ text }}</time>
  <span v-else class="text-base-content/40">&ndash;</span>
</template>

<script lang="ts" setup>
import { uptime, type Container } from "@/models/Container";

const { container } = defineProps<{
  container: Pick<Container, "state" | "startedAt" | "finishedAt">;
}>();

const { t } = useI18n();

const status = computed(() => uptime(container));

// Up and paused read as a span of time ("Up 8 minutes"), the stopped states as a moment
// ("Exited 3 hours ago"), the way docker ps words them. Same shared tick as RelativeTime.
const text = computed(() => {
  relativeTimeTick.value;
  if (!status.value) return "";
  const lang = locale.value === "" ? undefined : locale.value;
  const { state, since } = status.value;
  if (state === "up" || state === "paused") {
    return t(`container-status.${state}`, { duration: toElapsed(since, lang) });
  }
  return t(`container-status.${state}`, { time: toRelativeTime(since, lang) });
});
</script>
