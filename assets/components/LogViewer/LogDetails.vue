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
    <tbody ref="sortable">
      <tr v-for="(value, name) in entry.message" :key="name" class="hover">
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
const sortable = ref<HTMLElement | null>(null);

const container = currentContainer(toRef(() => entry.containerID));

const visibleKeys = persistentVisibleKeysForContainer(container);

const keys = ref(Object.keys(entry.message));

watchEffect(() => {
  console.log("keys", keys.value);
});

useSortable(sortable, keys);
</script>
