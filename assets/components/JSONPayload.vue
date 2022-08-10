<template>
  <ul class="fields" @click="expanded = !expanded">
    <li v-for="(value, name) in logEntry.filteredPayload?.value">
      <template v-if="value">
        <span class="has-text-grey">{{ name }}=</span><span class="has-text-weight-bold">{{ value }}</span>
      </template>
    </li>
  </ul>
  <field-list :fields="logEntry.entry.payload" :expanded="expanded" :visible-keys="visibleKeys"></field-list>
</template>
<script lang="ts" setup>
import { VisibleLogEntry } from "@/types/VisibleLogEntry";

import { PropType, ref } from "vue";

defineProps({
  logEntry: {
    type: Object as PropType<VisibleLogEntry>,
    required: true,
  },
  visibleKeys: {
    type: Array as PropType<string[][]>,
    default: [],
  },
});

const expanded = ref(false);
</script>

<style lang="scss" scoped>
.fields {
  display: inline-block;
  list-style: none;

  &:hover {
    cursor: pointer;
    &::after {
      content: "expand json";
      color: var(--secondary-color);
      display: inline-block;
      margin-left: 0.5em;
      font-family: "Segoe UI", Tahoma, Geneva, Verdana, sans-serif;
    }
  }

  li {
    display: inline-block;
    margin-left: 1em;
  }
}
</style>
