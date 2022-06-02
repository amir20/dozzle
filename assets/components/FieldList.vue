<template>
  <ul v-if="expanded">
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
        <a @click="toggleField(name)">add / remove </a>
        <span class="has-text-grey">{{ name }}=</span>[
        <span class="has-text-weight-bold" v-for="(item, index) in value">
          {{ item }}
          <span v-if="index !== value.length - 1">,</span>
        </span>
        ]
      </template>
      <template v-else>
        <a @click="toggleField(name)">add / remove </a>
        <span class="has-text-grey">{{ name }}=</span><span class="has-text-weight-bold">{{ value }}</span>
      </template>
    </li>
  </ul>
</template>
<script lang="ts" setup>
import { arrayEquals, isObject } from "@/utils";
import { PropType, toRaw } from "vue";

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

function toggleField(field: string) {
  const path = props.parentKey.concat(field);
  const index = props.visibleKeys.findIndex((keys) => arrayEquals(toRaw(keys), toRaw(path)));
  if (index > -1) {
    props.visibleKeys.splice(index, 1);
  } else {
    props.visibleKeys.push(path);
  }
}
</script>

<style lang="scss" scoped>
ul {
  margin-left: 2em;
}
</style>
