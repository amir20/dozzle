<template>
  <FieldList
    :fields="entry.unfilteredMessage"
    :expanded="true"
    :visible-keys="[]"
    v-if="entry instanceof ComplexLogEntry"
  />
</template>

<script setup lang="ts">
import { JSONObject, LogEntry, ComplexLogEntry } from "@/models/LogEntry";
const { entry } = defineProps<{ entry: LogEntry<string | JSONObject> }>();
</script>
