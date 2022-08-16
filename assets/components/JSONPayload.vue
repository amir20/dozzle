<template>
  <ul class="fields" @click="expanded = !expanded">
    <li v-for="(value, name) in logEntry.payload">
      <template v-if="value">
        <span class="has-text-grey">{{ name }}=</span>
        <span class="has-text-weight-bold" v-html="markSearch(value)"></span>
      </template>
    </li>
  </ul>
  <field-list :fields="logEntry.unfilteredPayload" :expanded="expanded" :visible-keys="visibleKeys"></field-list>
</template>
<script lang="ts" setup>
import { useSearchFilter } from "@/composables/search";
import { VisibleLogEntry } from "@/types/VisibleLogEntry";

import { PropType, ref } from "vue";

const { markSearch } = useSearchFilter();

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
