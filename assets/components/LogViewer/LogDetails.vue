<template>
  <div>
    <h1>{{ container.name }} - {{ entry.level }}</h1>
  </div>
  <table class="table" v-if="entry instanceof ComplexLogEntry">
    <thead class="text-lg">
      <tr>
        <th>Field</th>
        <th>Value</th>
        <th>Show</th>
      </tr>
    </thead>
    <tbody ref="list">
      <tr v-for="{ key, value, enabled } in fields" :key="key.join('.')" class="hover">
        <td class="font-mono">
          {{ key.join(".") }}
        </td>
        <td class="">
          <code>
            <span v-html="JSON.stringify(value)"></span>
          </code>
        </td>
        <td>
          <input type="checkbox" class="toggle toggle-primary" :checked="enabled" @change="toggleField(key)" />
        </td>
      </tr>
    </tbody>
  </table>
</template>
<style lang="postcss">
.font-mono {
  font-family:
    ui-monospace,
    SFMono-Regular,
    SF Mono,
    Consolas,
    Liberation Mono,
    monaco,
    Menlo,
    monospace;
}
</style>

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

function toggleField(key: string[]) {
  if (visibleKeys.value.size === 0) {
    visibleKeys.value = new Map<string[], boolean>(fields.value.map(({ key }) => [key, true]));
  }

  const enabled = visibleKeys.value.get(key);

  visibleKeys.value.set(key, !enabled);
}

const fields = computed({
  get() {
    const fieldsWithValue: { key: string[]; value: any; enabled: boolean }[] = [];
    if (visibleKeys.value.size === 0) {
      const map = flattenJSONToMap(entry.unfilteredMessage);

      for (const [key, value] of map) {
        fieldsWithValue.push({ key, value, enabled: true });
      }
    } else {
      for (const [key, enabled] of visibleKeys.value) {
        const value = getDeep(entry.unfilteredMessage, key);
        fieldsWithValue.push({ key, value, enabled });
      }
    }
    return fieldsWithValue;
  },
  set(value) {
    const map = new Map<string[], boolean>();
    for (const { key, enabled } of value) {
      map.set(key, enabled);
    }
    visibleKeys.value = map;
  },
});

useSortable(list, fields);
</script>
