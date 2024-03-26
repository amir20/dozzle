<template>
  <div class="group/item relative flex w-full gap-x-2" @click="expandToggle()">
    <label class="swap swap-rotate invisible absolute -left-4 top-0.5 size-4 group-hover/item:visible">
      <input type="checkbox" v-model="expanded" @click="expandToggle()" />
      <material-symbols:expand-all-rounded class="swap-off text-secondary" />
      <material-symbols:collapse-all-rounded class="swap-on text-secondary" />
    </label>
    <div v-if="showStd">
      <log-std :std="logEntry.std"></log-std>
    </div>
    <div v-if="showTimestamp">
      <log-date :date="logEntry.date"></log-date>
    </div>
    <div class="flex">
      <log-level :level="logEntry.level"></log-level>
    </div>
    <div>
      <ul class="fields cursor-pointer space-x-4" :class="{ expanded }">
        <li v-for="(value, name) in validValues">
          <span class="text-light">{{ name }}=</span><span class="font-bold" v-if="value === null">&lt;null&gt;</span>
          <template v-else-if="Array.isArray(value)">
            <span class="font-bold" v-html="markSearch(JSON.stringify(value))"> </span>
          </template>
          <span class="font-bold" v-html="markSearch(value)" v-else></span>
        </li>
        <li class="text-light" v-if="Object.keys(validValues).length === 0">all values are hidden</li>
      </ul>
      <field-list :fields="logEntry.unfilteredMessage" :expanded="expanded" :visible-keys="visibleKeys"></field-list>
    </div>
    <log-message-actions
      class="duration-250 absolute -right-1 opacity-0 transition-opacity delay-150 group-hover/entry:opacity-100"
      :message="() => JSON.stringify(logEntry.message)"
      :log-entry="logEntry"
    />
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

<style lang="postcss" scoped>
.text-light {
  @apply text-base-content/70;
}

.fields {
  li {
    @apply inline-flex;
  }
}
</style>
