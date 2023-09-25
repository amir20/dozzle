<template>
  <ul v-if="expanded" ref="root" class="ml-8">
    <li v-for="(value, name) in fields">
      <template v-if="isObject(value)">
        <span class="text-light">{{ name }}=</span>
        <field-list
          :fields="value"
          :parent-key="parentKey.concat(name)"
          :visible-keys="visibleKeys"
          expanded
        ></field-list>
      </template>
      <template v-else-if="Array.isArray(value)">
        <a @click="toggleField(name)" class="link-primary cursor-pointer">
          {{ hasField(name) ? "remove" : "add" }}&nbsp;</a
        >
        <span class="text-light">{{ name }}=</span>[
        <span class="font-bold" v-for="(item, index) in value">
          <span v-html="JSON.stringify(item)"></span><span v-if="index !== value.length - 1">,</span>
        </span>
        ]
      </template>
      <template v-else>
        <a @click="toggleField(name)" class="link-primary cursor-pointer">
          {{ hasField(name) ? "remove" : "add" }}&nbsp;</a
        >
        <span class="text-light">{{ name }}=</span><span class="font-bold" v-html="value"></span>
      </template>
    </li>
  </ul>
</template>
<script lang="ts" setup>
import { arrayEquals, isObject } from "@/utils";

const {
  fields,
  expanded = false,
  parentKey = [],
  visibleKeys = [],
} = defineProps<{
  fields: Record<string, any>;
  expanded?: boolean;
  parentKey?: string[];
  visibleKeys?: string[][];
}>();

const root = ref<HTMLElement>();

async function toggleField(field: string) {
  const index = fieldIndex(field);

  if (index > -1) {
    visibleKeys.splice(index, 1);
  } else {
    visibleKeys.push(parentKey.concat(field));
  }

  await nextTick();

  root.value?.scrollIntoView({
    block: "center",
  });
}

function hasField(field: string) {
  return fieldIndex(field) > -1;
}

function fieldIndex(field: string) {
  const path = parentKey.concat(field);
  return visibleKeys.findIndex((keys) => arrayEquals(toRaw(keys), toRaw(path)));
}
</script>

<style scoped lang="postcss">
.text-light {
  @apply text-base-content/70;
}
</style>
