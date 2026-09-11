<template>
  <!-- pr-24 keeps the title clear of the drawer's maximize/close buttons, which
       float over this slot. -->
  <header class="border-base-content/10 flex flex-wrap items-center gap-x-3 gap-y-2 border-b pr-24 pb-4">
    <span class="level-pill" :data-pill-level="entry.level ?? 'unknown'" v-if="entry.level">{{ entry.level }}</span>
    <h1 class="font-mono text-base tabular-nums max-md:hidden">
      <DateTime :date="entry.date" />
    </h1>
    <h2 class="text-base-content/55 text-xs">
      <RelativeTime :date="entry.date" />
      <span class="px-1">&middot;</span>
      <span :data-std="entry.std">{{ $t("log-details.on-std", { std: entry.std }) }}</span>
    </h2>
  </header>

  <div class="mt-5 flex flex-col gap-6">
    <!-- Facts about where the line came from. Small labels, plain values: this
         is context for the payload below, not the headline. -->
    <section class="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-3">
      <div class="min-w-0">
        <div class="field-label">{{ $t("label.container-name") }}</div>
        <div class="truncate font-medium" :title="container.name">{{ container.name }}</div>
      </div>
      <div class="min-w-0">
        <div class="field-label">{{ $t("label.host") }}</div>
        <div class="truncate font-medium" :title="hosts[container.host].name">{{ hosts[container.host].name }}</div>
      </div>
      <div class="min-w-0">
        <div class="field-label">{{ $t("log-details.image") }}</div>
        <div class="truncate font-medium" :title="container.image">{{ container.image }}</div>
      </div>
    </section>

    <section class="flex flex-col gap-2">
      <div class="flex items-center gap-1">
        <div class="field-label">{{ $t("log-details.raw-json") }}</div>

        <button
          class="icon-btn swap ml-auto outline-hidden"
          @click="copy(entry.rawMessage, { quiet: true })"
          :class="{ 'hover:swap-active': copied }"
          :title="$t('log-details.copy')"
        >
          <mdi:check class="swap-on" />
          <material-symbols:content-copy class="swap-off" />
        </button>

        <button class="icon-btn outline-hidden" @click="downloadJSON()" :title="$t('log-details.download')">
          <material-symbols:download />
        </button>
      </div>
      <div class="bg-base-200 border-base-content/10 max-h-125 overflow-auto rounded-md border p-3">
        <JsonFormatted :value="entry.rawMessage" class="text-xs leading-relaxed" />
      </div>
    </section>

    <section class="flex flex-col gap-2" v-if="entry instanceof ComplexLogEntry">
      <div class="flex items-center gap-3">
        <div class="field-label">{{ $t("log-details.fields") }}</div>
        <p class="text-base-content/45 text-xs">{{ $t("log-details.fields-hint") }}</p>
      </div>
      <table class="w-full table-fixed border-collapse text-sm">
        <thead>
          <tr class="border-base-content/15 border-b">
            <th class="field-label w-1/3 pb-1.5 text-left">{{ $t("log-details.field") }}</th>
            <th class="field-label pb-1.5 text-left max-md:hidden">{{ $t("log-details.value") }}</th>
            <th class="w-14 pb-1.5 text-right">
              <input
                type="checkbox"
                class="toggle toggle-primary toggle-xs align-middle"
                v-model="toggleAllFields"
                :title="$t('log-details.toggle-all')"
              />
            </th>
          </tr>
        </thead>
        <tbody ref="list">
          <tr v-for="{ key, value, enabled } in fields" :key="key.join('.')" class="field-row">
            <td class="cursor-move py-1.5 pr-3 font-mono break-all">
              <mdi:drag-vertical class="drag-handle -ml-1 inline size-4 align-middle" />
              <span :class="{ 'opacity-40': !enabled }">{{ key.join(".") }}</span>
            </td>
            <td class="text-base-content/65 truncate py-1.5 pr-3 font-mono max-md:hidden">
              <code :title="JSON.stringify(value)">{{ JSON.stringify(value) }}</code>
            </td>
            <td class="py-1.5 text-right">
              <input
                type="checkbox"
                class="toggle toggle-primary toggle-xs align-middle"
                :checked="enabled"
                @change="toggleField(key)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ComplexLogEntry } from "@/models/LogEntry";

const { entry } = defineProps<{ entry: ComplexLogEntry }>();
const { copy, copied } = useCopy();
const { currentContainer } = useContainerStore();
const list = ref<HTMLElement>();
const container = currentContainer(toRef(() => entry.containerID));
const visibleKeys = persistentVisibleKeysForContainer(container);
const { hosts } = useHosts();

const { useSortable } = await import("@vueuse/integrations/useSortable");

function downloadJSON() {
  let content = entry.rawMessage;
  try {
    content = JSON.stringify(JSON.parse(content), null, 2);
  } catch {
    // not valid JSON, download as is
  }

  const name = (container.value?.name ?? "log").replace(/[^\w.-]+/g, "-");
  const timestamp = entry.date.toISOString().replace(/[:.]/g, "-");

  const url = URL.createObjectURL(new Blob([content], { type: "application/json" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = `${name}-${timestamp}.json`;
  link.click();
  URL.revokeObjectURL(url);
}

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

useSortable(list, fields);
</script>

<style scoped>
@reference "@/main.css";

/* One label style for every section heading and column header, so the eye has
   a single "this is a label, not content" cue down the whole drawer. */
.field-label {
  @apply text-base-content/50 text-[0.7rem] font-semibold tracking-wider uppercase;
}

/* Keyed on data-pill-level, NOT data-level: LogLevel.vue ships an unscoped
   `[data-level="error"] { @apply !bg-red }` that would paint this solid and
   make the tint below unwinnable. */
.level-pill {
  background-color: color-mix(in oklab, var(--pill) 18%, transparent);
  color: var(--pill);
  border: 1px solid color-mix(in oklab, var(--pill) 40%, transparent);
  @apply inline-flex shrink-0 items-center rounded px-2 py-0.5 text-[0.7rem] font-bold tracking-wider uppercase;
  --pill: var(--color-base-content);
}
.level-pill[data-pill-level="debug"],
.level-pill[data-pill-level="trace"] {
  --pill: var(--color-purple);
}
.level-pill[data-pill-level="info"] {
  --pill: var(--color-green);
}
.level-pill[data-pill-level="warn"] {
  --pill: var(--color-orange);
}
.level-pill[data-pill-level="error"],
.level-pill[data-pill-level="fatal"] {
  --pill: var(--color-red);
}

[data-std="stdout"] {
  @apply text-blue;
}
[data-std="stderr"] {
  @apply text-red;
}

/* The handle only appears on the row being pointed at: eighteen of them stacked
   down the table read as noise, and the rows are draggable either way. */
.drag-handle {
  @apply text-base-content/40 opacity-0 transition-opacity;
}
.field-row {
  @apply border-base-content/10 border-b transition-colors;
}
.field-row:hover {
  background-color: color-mix(in oklab, var(--color-base-content) 6%, transparent);
}
.field-row:hover .drag-handle {
  @apply opacity-100;
}
</style>
