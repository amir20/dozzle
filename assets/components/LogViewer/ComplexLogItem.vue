<template>
  <div class="columns is-1 is-variable is-mobile">
    <div class="column is-narrow" v-if="showStd">
      <log-std :std="logEntry.std"></log-std>
    </div>
    <div class="column is-narrow" v-if="showTimestamp">
      <log-date :date="logEntry.date"></log-date>
    </div>
    <div class="column is-narrow is-flex">
      <log-level :level="logEntry.level"></log-level>
    </div>
    <div class="column">
      <ul class="fields" :class="{ expanded }" @click="expandToggle()">
        <li v-for="(value, name) in validValues">
          <span class="has-text-grey">{{ name }}=</span
          ><span class="has-text-weight-bold" v-if="value === null">&lt;null&gt;</span>
          <template v-else-if="Array.isArray(value)">
            <span class="has-text-weight-bold" v-html="markSearch(JSON.stringify(value))"> </span>
          </template>
          <span class="has-text-weight-bold" v-html="markSearch(value)" v-else></span>
        </li>
        <li class="has-text-grey" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
      </ul>
      <field-list :fields="logEntry.unfilteredMessage" :expanded="expanded" :visible-keys="visibleKeys"></field-list>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { type ComplexLogEntry } from "@/models/LogEntry";

const { markSearch } = useSearchFilter();

const { logEntry } = defineProps<{
  logEntry: ComplexLogEntry;
  visibleKeys: string[][];
}>();

const [expanded, expandToggle] = useToggle();

const validValues = computed(() => {
  return Object.fromEntries(Object.entries(logEntry.message).filter(([_, value]) => value !== undefined));
});
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

  &.expanded:hover {
    &::after {
      content: "collapse json";
    }
  }

  li {
    display: inline-block;

    & + li {
      margin-left: 1em;
    }
  }
}
</style>
