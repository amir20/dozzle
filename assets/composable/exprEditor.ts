import type { Completion } from "@codemirror/autocomplete";

export interface ExprEditorOptions {
  parent: HTMLElement;
  placeholder: string;
  initialValue: string;
  getHints: () => Completion[];
  onChange?: (value: string) => void;
}

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

export function createContainerHints(
  containerNames: string[],
  imageNames: string[],
  hostNames: string[],
): Completion[] {
  return [
    { label: "name", detail: "container name", type: "property" },
    { label: "id", detail: "container ID", type: "property" },
    { label: "image", detail: "container image", type: "property" },
    { label: "state", detail: "running, exited, etc.", type: "property" },
    { label: "health", detail: "healthy, unhealthy, none", type: "property" },
    { label: "host", detail: "docker host", type: "property" },
    { label: "labels", detail: "container labels map", type: "property" },
    ...exprOperators,
    { label: '"running"', detail: "state value", type: "string" },
    { label: '"exited"', detail: "state value", type: "string" },
    { label: '"created"', detail: "state value", type: "string" },
    { label: '"paused"', detail: "state value", type: "string" },
    { label: '"healthy"', detail: "health value", type: "string" },
    { label: '"unhealthy"', detail: "health value", type: "string" },
    { label: '"none"', detail: "health value", type: "string" },
    ...containerNames.map((name) => ({ label: `"${name}"`, detail: "container name", type: "string" }) as Completion),
    ...imageNames.map((image) => ({ label: `"${image}"`, detail: "image name", type: "string" }) as Completion),
    ...hostNames.map((host) => ({ label: `"${host}"`, detail: "host name", type: "string" }) as Completion),
  ];
}

export function createLogHints(messageKeys?: string[]): Completion[] {
  return [
    { label: "message", detail: "log message content", type: "property" },
    { label: "level", detail: "log level", type: "property" },
    { label: "stream", detail: "stdout or stderr", type: "property" },
    { label: "type", detail: "log type", type: "property" },
    { label: "timestamp", detail: "unix timestamp", type: "property" },
    { label: "id", detail: "log entry ID", type: "property" },
    ...(messageKeys ?? []).map(
      (key) => ({ label: `message.${key}`, detail: "message field", type: "property" }) as Completion,
    ),
    ...exprOperators,
    { label: '"error"', detail: "level value", type: "string" },
    { label: '"warn"', detail: "level value", type: "string" },
    { label: '"info"', detail: "level value", type: "string" },
    { label: '"debug"', detail: "level value", type: "string" },
    { label: '"trace"', detail: "level value", type: "string" },
    { label: '"stdout"', detail: "stream value", type: "string" },
    { label: '"stderr"', detail: "stream value", type: "string" },
    { label: 'level == "error"', detail: "match error logs", type: "text", boost: 10 },
    { label: 'message contains ""', detail: "search in message", type: "text", boost: 10 },
    { label: 'stream == "stderr"', detail: "match stderr", type: "text", boost: 10 },
  ];
}

export function createMetricHints(): Completion[] {
  return [
    { label: "cpu", detail: "CPU usage percent", type: "property" },
    { label: "memory", detail: "memory usage percent", type: "property" },
    { label: "memoryUsage", detail: "memory usage bytes", type: "property" },
    ...exprOperators,
    { label: ">", detail: "greater than", type: "operator" },
    { label: "<", detail: "less than", type: "operator" },
    { label: ">=", detail: "greater or equal", type: "operator" },
    { label: "<=", detail: "less or equal", type: "operator" },
    { label: "cpu > 80", detail: "CPU over 80%", type: "text", boost: 10 },
    { label: "memory > 90", detail: "memory over 90%", type: "text", boost: 10 },
    { label: "cpu > 80 || memory > 90", detail: "CPU or memory high", type: "text", boost: 10 },
  ];
}

function createAutocomplete(getHints: () => Completion[]) {
  return (context: any) => {
    const word = context.matchBefore(/[\w"=!&|]+/);
    if (!word && !context.explicit) return null;

    const currentWord = word ? word.text.toLowerCase() : "";
    const hints = getHints();
    const filtered = currentWord ? hints.filter((h) => h.label.toLowerCase().includes(currentWord)) : hints;

    return { from: word ? word.from : context.pos, options: filtered };
  };
}

export async function createExprEditor(options: ExprEditorOptions) {
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

  const state = EditorState.create({
    doc: options.initialValue,
    extensions: [
      EditorView.lineWrapping,
      placeholder(options.placeholder),
      autocompletion({
        override: [createAutocomplete(options.getHints)],
        activateOnTyping: true,
      }),
      keymap.of(completionKeymap),
      editorTheme,
      syntaxHighlighting(highlightStyle),
      EditorView.updateListener.of((update) => {
        if (update.docChanged && options.onChange) {
          options.onChange(update.view.state.doc.toString());
        }
      }),
    ],
  });

  return new EditorView({ state, parent: options.parent });
}
