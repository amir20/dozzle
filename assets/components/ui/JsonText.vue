<template>
  <template v-if="!highlight">{{ text }}</template>
  <template v-else>
    <template v-for="(part, i) in parts" :key="i">
      <mark v-if="part.match" class="bg-warning text-warning-content rounded px-0.5">{{ part.text }}</mark>
      <template v-else>{{ part.text }}</template>
    </template>
  </template>
</template>

<script lang="ts" setup>
const { text, highlight } = defineProps<{
  text: string;
  highlight?: string;
}>();

const parts = computed(() => {
  if (!highlight) return [{ text, match: false }];
  const pattern = highlight.replace(/[-/\\^$*+?.()|[\]{}]/g, "\\$&");
  const re = new RegExp(pattern, "gi");
  const result: { text: string; match: boolean }[] = [];
  let last = 0;
  for (const m of text.matchAll(re)) {
    if (m.index! > last) result.push({ text: text.slice(last, m.index), match: false });
    result.push({ text: m[0], match: true });
    last = m.index! + m[0].length;
  }
  if (last < text.length) result.push({ text: text.slice(last), match: false });
  return result;
});
</script>
