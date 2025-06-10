<template>
  <header class="flex items-center gap-4">
    <Tag :data-level="entry.level" class="text-white uppercase" v-if="entry.level">{{ entry.level }}</Tag>
    <h1 class="text-lg max-md:hidden">
      <DateTime :date="entry.date" />
    </h1>
    <h2 class="text-sm"><RelativeTime :date="entry.date" /> on {{ entry.std }}</h2>
  </header>

  <div class="mt-8 flex flex-col gap-10">
    <section class="grid grid-cols-3 gap-2">
      <div>
        <div class="font-thin">Container Name</div>
        <div class="truncate text-lg font-bold">{{ container.name }}</div>
      </div>
      <div>
        <div class="font-thin">Host</div>
        <div class="truncate text-lg font-bold">
          {{ hosts[container.host].name }}
        </div>
      </div>
      <div>
        <div class="font-thin">Image</div>
        <div class="truncate text-lg font-bold">{{ container.image }}</div>
      </div>
    </section>

    <section class="flex flex-col gap-2">
      <div class="flex gap-2">
        Raw JSON

        <UseClipboard v-slot="{ copy, copied }" :source="entry.rawMessage">
          <button class="swap outline-hidden" @click="copy()" :class="{ 'hover:swap-active': copied }">
            <mdi:check class="swap-on" />
            <material-symbols:content-copy class="swap-off" />
          </button>
        </UseClipboard>
      </div>
      <div class="bg-base-200 max-h-48 overflow-scroll rounded-sm border border-white/20 p-2">
        <pre v-html="syntaxHighlight(entry.rawMessage)"></pre>
      </div>
    </section>
    <table class="table-pin-rows table table-fixed" v-if="entry instanceof ComplexLogEntry">
      <caption class="caption-bottom">
        Fields are sortable by dragging and dropping.
      </caption>
      <thead class="text-lg">
        <tr>
          <th class="w-60">Field</th>
          <th class="max-md:hidden">Value</th>
          <th class="w-20">
            <input type="checkbox" class="toggle toggle-primary" v-model="toggleAllFields" title="Toggle all" />
          </th>
        </tr>
      </thead>
      <tbody ref="list">
        <tr v-for="{ key, value, enabled } in fields" :key="key.join('.')" class="hover">
          <td class="cursor-move font-mono break-all">
            {{ key.join(".") }}
          </td>
          <td class="truncate max-md:hidden">
            <code>{{ JSON.stringify(value) }}</code>
          </td>
          <td>
            <input type="checkbox" class="toggle toggle-primary" :checked="enabled" @change="toggleField(key)" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { ComplexLogEntry } from "@/models/LogEntry";
import { UseClipboard } from "@vueuse/components";

const { entry } = defineProps<{ entry: ComplexLogEntry }>();
const { currentContainer } = useContainerStore();
const list = ref<HTMLElement>();
const container = currentContainer(toRef(() => entry.containerID));
const visibleKeys = persistentVisibleKeysForContainer(container);
const { hosts } = useHosts();

const { useSortable } = await import("@vueuse/integrations/useSortable");

function toggleField(key: string[]) {
  if (visibleKeys.value.size === 0) {
    visibleKeys.value = new Map<string[], boolean>(fields.value.map(({ key }) => [key, true]));
  }

  const enabled = visibleKeys.value.get(key) ?? true;

  visibleKeys.value.set(key, !enabled);
}

const fields = computed({
  get() {
    const fieldsWithValue: { key: string[]; value: any; enabled: boolean }[] = [];
    const rawFields = JSON.parse(entry.rawMessage);
    const allFields = flattenJSONToMap(rawFields);
    if (visibleKeys.value.size === 0) {
      for (const [key, value] of allFields) {
        fieldsWithValue.push({ key, value, enabled: true });
      }
    } else {
      for (const [key, enabled] of visibleKeys.value) {
        const value = getDeep(rawFields, key);
        fieldsWithValue.push({ key, value, enabled });
      }

      for (const [key, value] of allFields) {
        if ([...visibleKeys.value.keys()].findIndex((k) => arrayEquals(k, key)) === -1) {
          fieldsWithValue.push({ key, value, enabled: true });
        }
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

const toggleAllFields = computed({
  get: () => fields.value.every(({ enabled }) => enabled),
  set(value) {
    if (visibleKeys.value.size === 0) {
      visibleKeys.value = new Map<string[], boolean>(fields.value.map(({ key }) => [key, true]));
    }
    for (const key of visibleKeys.value.keys()) {
      visibleKeys.value.set(key, value);
    }

    for (const field of fields.value) {
      visibleKeys.value.set(field.key, value);
    }
  },
});

function syntaxHighlight(json: string) {
  json = JSON.stringify(JSON.parse(json.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")), null, 2);
  return json.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|\b\d+\b)/g,
    function (match: string) {
      var cls = "json-number";
      if (match.startsWith('"')) {
        if (match.endsWith(":")) {
          cls = "json-key";
        } else {
          cls = "json-string";
        }
      } else if (/true|false/.test(match)) {
        cls = "json-boolean";
      } else if (/null/.test(match)) {
        cls = "json-null";
      }
      return `<span class="${cls}">${match}</span>`;
    },
  );
}

useSortable(list, fields);
</script>
<style scoped>
@reference "@/main.css";
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

pre {
  & :deep(.json-key) {
    @apply text-blue;
  }
  & :deep(.json-string) {
    @apply text-green;
  }
  & :deep(.json-number) {
    @apply text-orange;
  }
  & :deep(.json-boolean) {
    @apply text-purple;
  }
  & :deep(.json-null) {
    @apply text-red;
  }
}
</style>
