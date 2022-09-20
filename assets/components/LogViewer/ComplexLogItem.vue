<template>
  <div class="columns is-1 is-variable">
    <div class="column is-narrow" v-if="showTimestamp">
      <log-date :date="logEntry.date"></log-date>
    </div>
    <div class="column">
      <ul class="fields" @click="expanded = !expanded">
        <li v-for="(value, name) in logEntry.message">
          <template v-if="value">
            <span class="has-text-grey">{{ name }}=</span>
            <span class="has-text-weight-bold" v-html="markSearch(value)"></span>
          </template>
        </li>
      </ul>
      <field-list :fields="logEntry.unfilteredMessage" :expanded="expanded" :visible-keys="visibleKeys"></field-list>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { type ComplexLogEntry } from "@/models/LogEntry";

const { markSearch } = useSearchFilter();

defineProps<{
  logEntry: ComplexLogEntry;
  visibleKeys: string[][];
}>();

let expanded = $ref(false);
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
  }
}
</style>
