<template>
  <span class="json" :class="{ 'json-block': block }">
    <JsonValue :value="parsed" :indent="block ? 0 : -1" :highlight="highlight" />
  </span>
</template>

<script lang="ts" setup>
import JsonValue from "./JsonValue.vue";

const {
  value,
  highlight,
  block = true,
} = defineProps<{
  value: unknown;
  highlight?: string;
  block?: boolean;
}>();

const parsed = computed(() => {
  if (typeof value !== "string") return value;
  const trimmed = value.trim();
  if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) return value;
  try {
    return JSON.parse(trimmed);
  } catch {
    return value;
  }
});
</script>

<style scoped>
@reference "@/main.css";
.json-block {
  @apply block font-mono break-all whitespace-pre-wrap;
}
</style>
