<template>
  <div class="space-y-6 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">{{ isEditing ? "Edit Alert" : "Create Alert" }}</h2>
      <p class="text-base-content/60">Subscribe to log events matching your criteria</p>
    </div>

    <!-- Alert Name -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Alert Name</legend>
      <input
        ref="alertNameInput"
        v-model="alertName"
        type="text"
        class="input focus:input-primary w-full text-base"
        :class="alertName.trim() ? 'input-primary' : ''"
        required
        placeholder="e.g., Test API Errors"
      />
    </fieldset>

    <!-- Container Filter -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Container Filter</legend>
      <div
        class="input focus-within:input-primary w-full"
        :class="
          containerExpression.trim() && !containerResult?.error
            ? 'input-primary'
            : { 'input-error!': containerResult?.error }
        "
      >
        <div ref="containerEditorRef" class="w-full"></div>
      </div>
      <div v-if="containerResult" class="fieldset-label">
        <span v-if="containerResult.error" class="text-error">{{ containerResult.error }}</span>
        <span v-else-if="containerResult.containers?.length" class="text-success">
          <mdi:check class="inline" />
          {{ containerResult.containers.length }} containers match:
          {{ containerResult.containers.map((c) => c.name).join(", ") }}
        </span>
        <span v-else class="text-warning">
          <mdi:alert class="inline" />
          No containers match this filter
        </span>
      </div>
    </fieldset>

    <!-- Log Filter -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Log Filter</legend>
      <div
        class="input focus-within:input-primary w-full"
        :class="logExpression.trim() && !logError ? 'input-primary' : { 'input-error!': logError }"
      >
        <div ref="logEditorRef" class="w-full"></div>
      </div>
      <div v-if="logError || logExpression" class="fieldset-label">
        <span v-if="logError" class="text-error">{{ logError }}</span>
        <span v-else-if="logMessages.length" class="text-success">
          <mdi:check class="inline" />
          {{ logTotalCount }} logs match
        </span>
        <span v-else-if="!isLoading" class="text-warning">
          <mdi:alert class="inline" />
          No logs match this filter
        </span>
      </div>
    </fieldset>

    <!-- Destination -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Destination</legend>
      <details class="dropdown w-full" ref="destinationDropdown">
        <summary class="btn btn-outline w-full justify-between" :class="{ 'btn-primary': selectedDestination }">
          <span class="flex items-center gap-2">
            <template v-if="selectedDestination">
              <mdi:webhook v-if="selectedDestination.type === 'webhook'" />
              <mdi:cloud v-else />
              {{ selectedDestination.name }}
            </template>
            <span v-else class="text-base-content/60">Select a destination</span>
          </span>
          <carbon:caret-down />
        </summary>
        <ul class="dropdown-content menu bg-base-200 rounded-box z-50 mt-1 w-full border p-2 shadow-sm">
          <li v-for="dest in destinations" :key="dest.id">
            <a @click="selectDestination(dest.id)" :class="{ active: dispatcherId === dest.id }">
              <mdi:webhook v-if="dest.type === 'webhook'" />
              <mdi:cloud v-else />
              {{ dest.name }}
            </a>
          </li>
        </ul>
      </details>
      <div v-if="!destinations.length" class="fieldset-label">
        <span class="text-warning">
          <mdi:alert class="inline" />
          No destinations configured. Add one first.
        </span>
      </div>
    </fieldset>

    <!-- Log Preview -->
    <div v-if="logMessages.length" class="mt-4">
      <div class="mb-2 text-lg">Preview</div>
      <LogList
        :messages="logMessages"
        :last-selected-item="undefined"
        class="border-base-content/50 h-64 overflow-hidden rounded-lg border"
      />
    </div>

    <!-- Error -->
    <div v-if="saveError" class="alert alert-error">
      <span>{{ saveError }}</span>
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">Cancel</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveAlert">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? "Save" : "Create Alert" }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Completion } from "@codemirror/autocomplete";
import { type LogEvent, type LogEntry, type LogMessage, asLogEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import type { ContainerJson } from "@/types/Container";

import { useQuery, useMutation } from "@urql/vue";
import {
  GetDispatchersDocument,
  CreateNotificationRuleDocument,
  ReplaceNotificationRuleDocument,
  PreviewExpressionDocument,
} from "@/types/graphql";

export interface AlertData {
  id: number;
  name: string;
  containerExpression: string;
  logExpression: string;
  dispatcherId: number;
}

const { close, onCreated, alert } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  alert?: AlertData;
}>();

const isEditing = computed(() => !!alert);

// GraphQL queries and mutations
const dispatchersQuery = useQuery({ query: GetDispatchersDocument });
const createMutation = useMutation(CreateNotificationRuleDocument);
const replaceMutation = useMutation(ReplaceNotificationRuleDocument);
const previewMutation = useMutation(PreviewExpressionDocument);

const destinations = computed(() => dispatchersQuery.data.value?.dispatchers ?? []);

const containerEditorRef = ref<HTMLElement>();
const logEditorRef = ref<HTMLElement>();

const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore);

const containerNames = computed(() => [
  ...new Set(containers.value.filter((c) => c.state === "running").map((c) => c.name)),
]);
const imageNames = computed(() => [...new Set(containers.value.map((c) => c.image))]);
const hostNames = computed(() => [...new Set(containers.value.map((c) => c.host))]);

// Common operators for expr language
const exprOperators: Completion[] = [
  { label: "==", detail: "equals", type: "operator" },
  { label: "!=", detail: "not equals", type: "operator" },
  { label: "contains", detail: "string contains", type: "keyword" },
  { label: "startsWith", detail: "string starts with", type: "keyword" },
  { label: "endsWith", detail: "string ends with", type: "keyword" },
  { label: "matches", detail: "regex match", type: "keyword" },
  { label: "&&", detail: "logical AND", type: "operator" },
  { label: "||", detail: "logical OR", type: "operator" },
  { label: "!", detail: "logical NOT", type: "operator" },
  { label: "in", detail: "membership test", type: "keyword" },
  { label: "not in", detail: "negative membership", type: "keyword" },
];

function createContainerHints(): Completion[] {
  const hints: Completion[] = [
    // Fields
    { label: "name", detail: "container name", type: "property" },
    { label: "id", detail: "container ID", type: "property" },
    { label: "image", detail: "container image", type: "property" },
    { label: "state", detail: "running, exited, etc.", type: "property" },
    { label: "health", detail: "healthy, unhealthy, none", type: "property" },
    { label: "host", detail: "docker host", type: "property" },
    { label: "labels", detail: "container labels map", type: "property" },

    ...exprOperators,

    // State values
    { label: '"running"', detail: "state value", type: "string" },
    { label: '"exited"', detail: "state value", type: "string" },
    { label: '"created"', detail: "state value", type: "string" },
    { label: '"paused"', detail: "state value", type: "string" },

    // Health values
    { label: '"healthy"', detail: "health value", type: "string" },
    { label: '"unhealthy"', detail: "health value", type: "string" },
    { label: '"none"', detail: "health value", type: "string" },
  ];

  // Add dynamic container names
  for (const name of containerNames.value) {
    hints.push({ label: `"${name}"`, detail: "container name", type: "string" });
  }

  // Add dynamic image names
  for (const image of imageNames.value) {
    hints.push({ label: `"${image}"`, detail: "image name", type: "string" });
  }

  // Add dynamic host names
  for (const host of hostNames.value) {
    hints.push({ label: `"${host}"`, detail: "host name", type: "string" });
  }

  return hints;
}

function createLogHints(): Completion[] {
  return [
    // Fields
    { label: "message", detail: "log message content", type: "property" },
    { label: "level", detail: "log level", type: "property" },
    { label: "stream", detail: "stdout or stderr", type: "property" },
    { label: "type", detail: "log type", type: "property" },
    { label: "timestamp", detail: "unix timestamp", type: "property" },
    { label: "id", detail: "log entry ID", type: "property" },

    ...exprOperators,

    // Level values
    { label: '"error"', detail: "level value", type: "string" },
    { label: '"warn"', detail: "level value", type: "string" },
    { label: '"warning"', detail: "level value", type: "string" },
    { label: '"info"', detail: "level value", type: "string" },
    { label: '"debug"', detail: "level value", type: "string" },
    { label: '"trace"', detail: "level value", type: "string" },

    // Stream values
    { label: '"stdout"', detail: "stream value", type: "string" },
    { label: '"stderr"', detail: "stream value", type: "string" },

    // Common snippets
    { label: 'level == "error"', detail: "match error logs", type: "text", boost: 10 },
    { label: 'message contains ""', detail: "search in message", type: "text", boost: 10 },
    { label: 'stream == "stderr"', detail: "match stderr", type: "text", boost: 10 },
  ];
}

function createAutocomplete(getHints: () => Completion[]) {
  return (context: any) => {
    const word = context.matchBefore(/[\w"=!&|]+/);

    if (!word && !context.explicit) return null;

    const currentWord = word ? word.text.toLowerCase() : "";
    const hints = getHints();

    const filtered = currentWord ? hints.filter((h) => h.label.toLowerCase().includes(currentWord)) : hints;

    return {
      from: word ? word.from : context.pos,
      options: filtered,
    };
  };
}

// Lazy load CodeMirror dependencies
const [
  { EditorView, keymap, placeholder },
  { EditorState },
  { autocompletion, completionKeymap },
  { HighlightStyle, syntaxHighlighting },
  { tags },
] = await Promise.all([
  import("@codemirror/view"),
  import("@codemirror/state"),
  import("@codemirror/autocomplete"),
  import("@codemirror/language"),
  import("@lezer/highlight"),
]);

// Theme using CSS variables that automatically adapt to light/dark mode
const editorTheme = EditorView.theme({
  "&": {
    backgroundColor: "var(--color-base-100)",
    color: "var(--color-base-content)",
  },
  ".cm-content": {
    caretColor: "var(--color-primary)",
  },
  ".cm-cursor": {
    borderLeftColor: "var(--color-primary)",
  },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
    backgroundColor: "var(--color-base-300)",
  },
  ".cm-activeLine": {
    backgroundColor: "color-mix(in oklch, var(--color-base-200) 50%, transparent)",
  },
  ".cm-gutters": {
    backgroundColor: "var(--color-base-200)",
    color: "color-mix(in oklch, var(--color-base-content) 50%, transparent)",
    border: "none",
  },
  ".cm-activeLineGutter": {
    backgroundColor: "var(--color-base-300)",
  },
});

// Syntax highlighting using CSS variables
const highlightStyle = HighlightStyle.define([
  { tag: tags.keyword, color: "var(--color-primary)" },
  { tag: tags.operator, color: "var(--color-secondary)" },
  { tag: tags.string, color: "var(--color-success)" },
  { tag: tags.number, color: "var(--color-warning)" },
  { tag: tags.bool, color: "var(--color-warning)" },
  { tag: tags.propertyName, color: "var(--color-info)" },
  { tag: tags.variableName, color: "var(--color-base-content)" },
  {
    tag: tags.comment,
    color: "color-mix(in oklch, var(--color-base-content) 50%, transparent)",
    fontStyle: "italic",
  },
]);

function createEditorState(
  getHints: () => Completion[],
  placeholderText: string,
  initialValue: string,
  onChange?: (value: string) => void,
) {
  return EditorState.create({
    doc: initialValue,
    extensions: [
      EditorView.lineWrapping,
      placeholder(placeholderText),
      autocompletion({
        override: [createAutocomplete(getHints)],
        activateOnTyping: true,
      }),
      keymap.of(completionKeymap),
      editorTheme,
      syntaxHighlighting(highlightStyle),
      EditorView.updateListener.of((update) => {
        if (update.docChanged && onChange) {
          onChange(update.view.state.doc.toString());
        }
      }),
    ],
  });
}

const containerEditorView = shallowRef<InstanceType<typeof EditorView>>();
const logEditorView = shallowRef<InstanceType<typeof EditorView>>();

const alertNameInput = ref<HTMLInputElement>();
const alertName = ref(alert?.name ?? "");
useFocus(alertNameInput, { initialValue: true });
const containerExpression = ref(alert?.containerExpression ?? "");
const logExpression = ref(alert?.logExpression ?? "");
const dispatcherId = ref(alert?.dispatcherId ?? 0);
const selectedDestination = computed(() => destinations.value.find((d) => d.id === dispatcherId.value));
const destinationDropdown = ref<HTMLDetailsElement>();

function selectDestination(id: number) {
  dispatcherId.value = id;
  destinationDropdown.value?.removeAttribute("open");
}

interface ContainerResult {
  error?: string;
  containers?: Container[];
}

const containerResult = ref<ContainerResult | null>(null);
const logError = ref<string | null>(null);
const logTotalCount = ref(0);
const logMessages = shallowRef<LogEntry<LogMessage>[]>([]);
const isLoading = ref(false);
const isSaving = ref(false);
const saveError = ref<string | null>(null);

const canSave = computed(() => {
  return (
    alertName.value.trim() &&
    containerExpression.value.trim() &&
    dispatcherId.value > 0 &&
    !containerResult.value?.error &&
    !logError.value &&
    !isSaving.value
  );
});

async function saveAlert() {
  if (!canSave.value) return;

  isSaving.value = true;
  saveError.value = null;

  try {
    const input = {
      name: alertName.value.trim(),
      containerExpression: containerExpression.value,
      logExpression: logExpression.value,
      dispatcherId: dispatcherId.value!,
      enabled: true,
    };

    const result = isEditing.value
      ? await replaceMutation.executeMutation({ id: alert!.id, input })
      : await createMutation.executeMutation({ input });

    if (result.error) {
      throw new Error(result.error.message);
    }

    onCreated?.();
    close?.();
  } catch (e) {
    saveError.value = e instanceof Error ? e.message : "Failed to save alert";
  } finally {
    isSaving.value = false;
  }
}

async function validateExpressions() {
  if (!containerExpression.value && !logExpression.value) {
    containerResult.value = null;
    logError.value = null;
    logTotalCount.value = 0;
    logMessages.value = [];
    return;
  }

  isLoading.value = true;

  try {
    const result = await previewMutation.executeMutation({
      input: {
        containerExpression: containerExpression.value,
        logExpression: logExpression.value || undefined,
      },
    });

    if (result.error) {
      throw new Error(result.error.message);
    }

    const data = result.data?.previewExpression;
    if (!data) {
      throw new Error("No data returned");
    }

    // Update container result
    if (containerExpression.value) {
      containerResult.value = {
        error: data.containerError ?? undefined,
        containers: data.matchedContainers?.map((c) => Container.fromJSON(c as ContainerJson)),
      };
    } else {
      containerResult.value = null;
    }

    // Update log result
    if (logExpression.value && !data.containerError) {
      logError.value = data.logError ?? null;
      logTotalCount.value = data.totalLogs;
      logMessages.value =
        data.matchedLogs?.map((event) =>
          asLogEntry({
            t: (event.type as LogEvent["t"]) ?? "single",
            m: event.message as LogEvent["m"],
            ts: event.timestamp,
            id: event.id,
            l: (event.level as LogEvent["l"]) ?? "unknown",
            s: (event.stream as LogEvent["s"]) ?? "unknown",
            c: "",
            rm: "",
          }),
        ) ?? [];
    } else {
      logError.value = null;
      logTotalCount.value = 0;
      logMessages.value = [];
    }
  } catch (e) {
    containerResult.value = {
      error: e instanceof Error ? e.message : "Unknown error",
    };
  } finally {
    isLoading.value = false;
  }
}

const debouncedValidate = useDebounceFn(validateExpressions, 500);

watch([containerExpression, logExpression], () => {
  isLoading.value = true;
  debouncedValidate();
});

onMounted(() => {
  if (containerEditorRef.value) {
    containerEditorView.value = new EditorView({
      state: createEditorState(createContainerHints, 'name contains "api"', alert?.containerExpression ?? "", (v) => {
        containerExpression.value = v;
      }),
      parent: containerEditorRef.value,
    });
  }

  if (logEditorRef.value) {
    logEditorView.value = new EditorView({
      state: createEditorState(
        createLogHints,
        'level == "error" && message contains "timeout"',
        alert?.logExpression ?? "",
        (v) => {
          logExpression.value = v;
        },
      ),
      parent: logEditorRef.value,
    });
  }
});

onScopeDispose(() => {
  containerEditorView.value?.destroy();
  logEditorView.value?.destroy();
});
</script>

<style scoped>
@reference "@/main.css";

:deep(.cm-editor.cm-focused) {
  outline: none;
}

:deep(.cm-scroller) {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace;
}
</style>

<style>
@reference "@/main.css";

/* Global styles for CodeMirror autocomplete tooltip (rendered outside component) */
.cm-tooltip {
  @apply bg-base-200! border-base-content/40! min-w-96 rounded-sm border shadow-md;
}

.cm-tooltip-autocomplete ul {
  @apply font-sans;
}

.cm-tooltip-autocomplete ul li {
  @apply my-1 px-2;
}

.cm-tooltip-autocomplete ul li[aria-selected] {
  @apply bg-primary/20 text-base-content!;
}

.cm-completionLabel {
  @apply text-base-content!;
}

.cm-completionDetail {
  @apply text-base-content/60! ml-2 italic;
}

.cm-completionMatchedText {
  @apply text-primary! font-bold no-underline;
}

.cm-completionIcon {
  @apply mr-2 opacity-70;
}
</style>
