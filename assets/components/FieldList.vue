<template>
  <ul v-if="expanded" ref="root">
    <li v-for="(value, name) in fields">
      <template v-if="isObject(value)">
        <span class="has-text-grey">{{ name }}=</span>
        <field-list
          :fields="value"
          :parent-key="parentKey.concat(name)"
          :visible-keys="visibleKeys"
          expanded
        ></field-list>
      </template>
      <template v-else-if="Array.isArray(value)">
        <a @click="toggleField(name)"> {{ hasField(name) ? "remove" : "add" }}&nbsp;</a>
        <span class="has-text-grey">{{ name }}=</span>[
        <span class="has-text-weight-bold" v-for="(item, index) in value">
          {{ item }}
          <span v-if="index !== value.length - 1">,</span>
        </span>
        ]
      </template>
      <template v-else>
        <a @click="toggleField(name)"> {{ hasField(name) ? "remove" : "add" }}&nbsp;</a>
        <span class="has-text-grey">{{ name }}=</span><span class="has-text-weight-bold">{{ value }}</span>
      </template>
    </li>
  </ul>
</template>
<script lang="ts" setup>
import { arrayEquals, isObject } from "@/utils";
import { nextTick, PropType, ref, toRaw } from "vue";

const props = defineProps({
  fields: {
    type: Object as PropType<Record<string, any>>,
    required: true,
  },
  expanded: {
    type: Boolean,
    default: false,
  },
  parentKey: {
    type: Array as PropType<string[]>,
    default: [],
  },
  visibleKeys: {
    type: Array as PropType<string[][]>,
    default: [],
  },
});

const root = ref<HTMLElement>();

async function toggleField(field: string) {
  const index = fieldIndex(field);

  if (index > -1) {
    props.visibleKeys.splice(index, 1);
  } else {
    props.visibleKeys.push(props.parentKey.concat(field));
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
  const path = props.parentKey.concat(field);
  return props.visibleKeys.findIndex((keys) => arrayEquals(toRaw(keys), toRaw(path)));
}
</script>

<style lang="scss" scoped>
ul {
  margin-left: 2em;
}
</style>
