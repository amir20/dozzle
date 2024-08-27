<template>
  <div>
    <h1>{{ container.name }} - {{ entry.level }}</h1>
  </div>
  <table class="table" v-if="entry instanceof ComplexLogEntry">
    <thead>
      <tr>
        <th>Field</th>
        <th>Value</th>
        <th>Show</th>
      </tr>
    </thead>
    <tbody ref="list">
      <tr v-for="[name, value] in fields" :key="name" class="hover">
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
import { ComplexLogEntry } from "@/models/LogEntry";
const { entry } = $defineProps<{ entry: ComplexLogEntry }>();
import { useSortable } from "@vueuse/integrations/useSortable";

const { currentContainer } = useContainerStore();

if (!(entry instanceof ComplexLogEntry)) {
  throw new Error("entry must be a ComplexLogEntry");
}
const list = ref<HTMLElement>();

const container = currentContainer(toRef(() => entry.containerID));

const visibleKeys = persistentVisibleKeysForContainer(container);

const fields = computed({
  get() {
    const all = flattenJSON(entry.unfilteredMessage);
    if (visibleKeys.value.length === 0) {
      return Object.entries(all);
    } else {
      return Object.entries(entry.message);
    }
  },
  set(value) {
    const keys = value.map(([key]) => key);
    console.log(keys);
    visibleKeys.value = keys.map((key) => key.split("."));
  },
});

useSortable(list, fields);
</script>
