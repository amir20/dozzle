<template>This is log detail.</template>

<script setup lang="ts">
import { JSONObject, LogEntry } from "@/models/LogEntry";
const { entry } = defineProps<{ entry: LogEntry<string | JSONObject> }>();
</script>
