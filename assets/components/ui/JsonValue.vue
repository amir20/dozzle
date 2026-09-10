<template>
  <template v-if="value === null">
    <span class="json-null">null</span>
  </template>
  <template v-else-if="typeof value === 'boolean'">
    <span class="json-boolean">{{ String(value) }}</span>
  </template>
  <template v-else-if="typeof value === 'number'">
    <span class="json-number">{{ value }}</span>
  </template>
  <template v-else-if="typeof value === 'string'">
    <span class="json-string">"<JsonText :text="value" :highlight="highlight" />"</span>
  </template>
  <template v-else-if="Array.isArray(value)">
    <template v-if="value.length === 0">
      <span>[]</span>
    </template>
    <template v-else-if="indent < 0">
      <span>[</span>
      <template v-for="(item, i) in value" :key="i">
        <JsonValue :value="item" :indent="indent" :highlight="highlight" />
        <span v-if="i < value.length - 1">, </span>
      </template>
      <span>]</span>
    </template>
    <template v-else>
      <span>[</span>
      <template v-for="(item, i) in value" :key="i">
        <span class="json-newline">{{ "\n" + pad(indent + 1) }}</span>
        <JsonValue :value="item" :indent="indent + 1" :highlight="highlight" />
        <span v-if="i < value.length - 1">,</span>
      </template>
      <span class="json-newline">{{ "\n" + pad(indent) }}</span>
      <span>]</span>
    </template>
  </template>
  <template v-else-if="typeof value === 'object'">
    <template v-if="entries.length === 0">
      <span>{}</span>
    </template>
    <template v-else-if="indent < 0">
      <span>{</span>
      <template v-for="([k, v], i) in entries" :key="k">
        <span class="json-key">"<JsonText :text="k" :highlight="highlight" />"</span><span>: </span>
        <JsonValue :value="v" :indent="indent" :highlight="highlight" />
        <span v-if="i < entries.length - 1">, </span>
      </template>
      <span>}</span>
    </template>
    <template v-else>
      <span>{</span>
      <template v-for="([k, v], i) in entries" :key="k">
        <span class="json-newline">{{ "\n" + pad(indent + 1) }}</span>
        <span class="json-key">"<JsonText :text="k" :highlight="highlight" />"</span><span>: </span>
        <JsonValue :value="v" :indent="indent + 1" :highlight="highlight" />
        <span v-if="i < entries.length - 1">,</span>
      </template>
      <span class="json-newline">{{ "\n" + pad(indent) }}</span>
      <span>}</span>
    </template>
  </template>
  <template v-else>
    <span>{{ String(value) }}</span>
  </template>
</template>

<script lang="ts" setup>
import JsonText from "./JsonText.vue";

const {
  value,
  indent = 0,
  highlight,
} = defineProps<{
  value: unknown;
  indent?: number;
  highlight?: string;
}>();

const entries = computed(() =>
  value && typeof value === "object" && !Array.isArray(value) ? Object.entries(value as Record<string, unknown>) : [],
);

function pad(level: number): string {
  return "  ".repeat(Math.max(level, 0));
}
</script>

<style scoped>
@reference "@/main.css";
.json-key {
  @apply text-blue;
}
.json-string {
  @apply text-green;
}
.json-number {
  @apply text-orange;
}
.json-boolean {
  @apply text-purple;
}
.json-null {
  @apply text-red;
}
</style>
