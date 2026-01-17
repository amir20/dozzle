<template>
  <PageWithLinks>
    <section>
      <div class="has-underline">
        <h2>Notifications</h2>
      </div>

      <div class="space-y-6">
        <div>
          <label class="label">Container Filter</label>
          <p class="text-base-content/70 mb-2 text-sm">
            Filter which containers to watch. Available fields: <code>name</code>, <code>id</code>, <code>image</code>,
            <code>state</code>, <code>health</code>, <code>host</code>, <code>labels</code>
          </p>
          <div class="editor-container">
            <div ref="containerEditorRef" class="editor"></div>
          </div>
        </div>

        <div>
          <label class="label">Log Filter</label>
          <p class="text-base-content/70 mb-2 text-sm">
            Filter which log entries trigger notifications. Available fields: <code>message</code>, <code>level</code>,
            <code>stream</code>, <code>type</code>, <code>timestamp</code>
          </p>
          <div class="editor-container">
            <div ref="logEditorRef" class="editor"></div>
          </div>
        </div>
      </div>
    </section>
  </PageWithLinks>
</template>

<script lang="ts" setup>
import { EditorView, keymap, placeholder } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { autocompletion, completionKeymap, type CompletionContext, type Completion } from "@codemirror/autocomplete";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags } from "@lezer/highlight";

const containerEditorRef = ref<HTMLElement>();
const logEditorRef = ref<HTMLElement>();

const containerStore = useContainerStore();
const { containers } = storeToRefs(containerStore);

const containerNames = computed(() => [...new Set(containers.value.map((c) => c.name))]);
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
  return (context: CompletionContext) => {
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
  { tag: tags.comment, color: "color-mix(in oklch, var(--color-base-content) 50%, transparent)", fontStyle: "italic" },
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

let containerEditorView: EditorView | null = null;
let logEditorView: EditorView | null = null;

const containerExpression = ref("");
const logExpression = ref("");

onMounted(() => {
  if (containerEditorRef.value) {
    containerEditorView = new EditorView({
      state: createEditorState(createContainerHints, 'name contains "api"', (v) => {
        containerExpression.value = v;
      }),
      parent: containerEditorRef.value,
    });
  }

  if (logEditorRef.value) {
    logEditorView = new EditorView({
      state: createEditorState(createLogHints, 'level == "error" && message contains "timeout"', (v) => {
        logExpression.value = v;
      }),
      parent: logEditorRef.value,
    });
  }
});

onUnmounted(() => {
  containerEditorView?.destroy();
  logEditorView?.destroy();
});
</script>

<style scoped>
@reference "@/main.css";

.has-underline {
  @apply border-base-content/50 mb-4 border-b py-2;

  h2 {
    @apply text-3xl;
  }
}

:deep(a:not(.menu a):not(.btn)) {
  @apply text-primary underline-offset-4 hover:underline;
}

.label {
  @apply text-base-content mb-1 block text-lg font-medium;
}

.editor-container {
  @apply border-base-content/20 rounded-lg border;
}

.editor :deep(.cm-editor) {
  @apply rounded-lg;
}

.editor :deep(.cm-editor.cm-focused) {
  outline: none;
}

.editor :deep(.cm-scroller) {
  @apply p-2;
}

code {
  @apply bg-base-300 rounded px-1 py-0.5 text-sm;
}
</style>

<style>
/* Global styles for CodeMirror autocomplete tooltip (rendered outside component) */
.cm-tooltip {
  background-color: var(--color-base-200) !important;
  border: 1px solid color-mix(in oklch, var(--color-base-content) 20%, transparent) !important;
  border-radius: 0.5rem;
  box-shadow:
    0 4px 6px -1px rgb(0 0 0 / 0.1),
    0 2px 4px -2px rgb(0 0 0 / 0.1);
}

.cm-tooltip-autocomplete ul {
  font-family: inherit;
}

.cm-tooltip-autocomplete ul li {
  padding: 0.25rem 0.5rem;
}

.cm-tooltip-autocomplete ul li[aria-selected] {
  background-color: color-mix(in oklch, var(--color-primary) 20%, transparent) !important;
  color: var(--color-base-content);
}

.cm-completionLabel {
  color: var(--color-base-content);
}

.cm-completionDetail {
  color: color-mix(in oklch, var(--color-base-content) 60%, transparent);
  margin-left: 0.5rem;
  font-style: italic;
}

.cm-completionMatchedText {
  color: var(--color-primary);
  text-decoration: none;
  font-weight: bold;
}

.cm-completionIcon {
  opacity: 0.7;
}
</style>
