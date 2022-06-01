<template>
  <ul v-if="expanded">
    <li v-for="(value, name) in fields">
      <template v-if="isObject(value)">
        <span class="has-text-grey">{{ name }}=</span>
        <FieldList :fields="value" :parent-key="parentKey.concat(name)" expanded></FieldList>
      </template>
      <template v-else-if="Array.isArray(value)">
        <span class="has-text-grey">{{ name }}=</span>[
        <span class="has-text-weight-bold" v-for="(item, index) in value">
          {{ item }}
          <span v-if="index !== value.length - 1">,</span>
        </span>
        ]
      </template>
      <template v-else>
        <span class="has-text-grey">{{ name }}=</span><span class="has-text-weight-bold">{{ value }}</span>
      </template>
    </li>
  </ul>
</template>
<script lang="ts" setup>
import { PropType } from "vue";

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
});

function isObject(value: any): value is Record<string, any> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
</script>

<style lang="scss" scoped>
ul {
  margin-left: 2em;
}
</style>
