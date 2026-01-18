<template>
  <div class="space-y-6 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">Create Alert</h2>
      <p class="text-base-content/60">Subscribe to log events matching your criteria</p>
    </div>

    <!-- Alert Name -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Alert Name</legend>
      <input
        v-model="alertName"
        type="text"
        class="input focus:input-primary w-full"
        placeholder="e.g., Test API Errors"
      />
    </fieldset>

    <!-- Container Filter -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Container Filter</legend>
      <div class="input focus-within:input-primary w-full" :class="{ 'input-error!': containerResult?.error }">
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
      <div class="input focus-within:input-primary w-full" :class="{ 'input-error!': logError }">
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
    <div v-if="createError" class="alert alert-error">
      <span>{{ createError }}</span>
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-2 pt-4">
      <button class="btn" @click="close?.()">Cancel</button>
      <button class="btn btn-primary" :disabled="!canCreate" @click="createAlert">
        <span v-if="isCreating" class="loading loading-spinner loading-sm"></span>
        Create Alert
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Completion } from "@codemirror/autocomplete";
import { type LogEvent, type LogEntry, type LogMessage, asLogEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";
import type { ContainerJson } from "@/types/Container";

const { close } = defineProps<{ close?: () => void }>();

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

function createEditorState(getHints: () => Completion[], placeholderText: string, onChange?: (value: string) => void) {
  return EditorState.create({
    doc: "",
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

const alertName = ref("");
const containerExpression = ref("");
const logExpression = ref("");

interface PreviewResponse {
  containerError?: string;
  logError?: string;
  matchedContainers?: ContainerJson[];
  matchedLogs?: LogEvent[];
  totalLogs: number;
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
const isCreating = ref(false);
const createError = ref<string | null>(null);

const canCreate = computed(() => {
  return (
    alertName.value.trim() &&
    containerExpression.value.trim() &&
    !containerResult.value?.error &&
    !logError.value &&
    !isCreating.value
  );
});

async function createAlert() {
  if (!canCreate.value) return;

  isCreating.value = true;
  createError.value = null;

  try {
    const response = await fetch(withBase("/api/notifications/subscriptions"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: alertName.value.trim(),
        containerExpression: containerExpression.value,
        logExpression: logExpression.value,
      }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `HTTP ${response.status}`);
    }

    close?.();
  } catch (e) {
    createError.value = e instanceof Error ? e.message : "Failed to create alert";
  } finally {
    isCreating.value = false;
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
    const response = await fetch(withBase("/api/notifications/preview"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        containerExpression: containerExpression.value,
        logExpression: logExpression.value,
      }),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const data: PreviewResponse = await response.json();

    // Update container result
    if (containerExpression.value) {
      containerResult.value = {
        error: data.containerError,
        containers: data.matchedContainers?.map(Container.fromJSON),
      };
    } else {
      containerResult.value = null;
    }

    // Update log result
    if (logExpression.value && !data.containerError) {
      logError.value = data.logError ?? null;
      logTotalCount.value = data.totalLogs;
      logMessages.value = data.matchedLogs?.map((event) => asLogEntry(event)) ?? [];
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
      state: createEditorState(createContainerHints, 'name contains "api"', (v) => {
        containerExpression.value = v;
      }),
      parent: containerEditorRef.value,
    });
    containerEditorView.value.focus();
  }

  if (logEditorRef.value) {
    logEditorView.value = new EditorView({
      state: createEditorState(createLogHints, 'level == "error" && message contains "timeout"', (v) => {
        logExpression.value = v;
      }),
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
