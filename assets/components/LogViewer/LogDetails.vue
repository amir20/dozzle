<template>
  <table class="table" v-if="entry instanceof ComplexLogEntry">
    <thead>
      <tr>
        <th>Field</th>
        <th>Value</th>
        <th>Show</th>
      </tr>
    </thead>
    <tbody ref="sortable">
      <tr v-for="{ name, value } in list" :key="name" class="hover">
        <td>
          {{ name }}
        </td>
        <td>
          {{ value }}
        </td>
        <td>
          <button class="btn btn-ghost btn-xs">details</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script setup lang="ts">
import { JSONObject, LogEntry, ComplexLogEntry } from "@/models/LogEntry";
const { entry } = defineProps<{ entry: LogEntry<string | JSONObject> }>();
import { useSortable } from "@vueuse/integrations/useSortable";

if (!(entry instanceof ComplexLogEntry)) {
  throw new Error("entry must be a ComplexLogEntry");
}
const sortable = ref<HTMLElement | null>(null);
const list = ref(Object.entries(entry.unfilteredMessage).map(([name, value]) => ({ name, value })));
useSortable(sortable, list);
</script>
