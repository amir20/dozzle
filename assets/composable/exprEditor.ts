import type { Completion, CompletionContext, CompletionSource } from "@codemirror/autocomplete";
import type { StringStream } from "@codemirror/language";

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
    ...containerNames.map((name): Completion => ({ label: `"${name}"`, detail: "container name", type: "string" })),
    ...imageNames.map((image): Completion => ({ label: `"${image}"`, detail: "image name", type: "string" })),
    ...hostNames.map((host): Completion => ({ label: `"${host}"`, detail: "host name", type: "string" })),
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
    ...(messageKeys ?? []).map((key): Completion => ({
      label: `message.${key}`,
      detail: "message field",
      type: "property",
    })),
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
    { label: "mounts", detail: "list of container mounts with free-space info", type: "property" },
    { label: ".usedPercent", detail: "mount field: % of mount used", type: "property" },
    { label: ".availableBytes", detail: "mount field: free bytes on mount", type: "property" },
    { label: ".destination", detail: "mount field: in-container mount path", type: "property" },
    { label: "any(mounts, ...)", detail: "true if any mount matches the predicate", type: "keyword" },
    ...exprOperators,
    { label: ">", detail: "greater than", type: "operator" },
    { label: "<", detail: "less than", type: "operator" },
    { label: ">=", detail: "greater or equal", type: "operator" },
    { label: "<=", detail: "less or equal", type: "operator" },
    { label: "cpu > 80", detail: "CPU over 80%", type: "text", boost: 10 },
    { label: "memory > 90", detail: "memory over 90%", type: "text", boost: 10 },
    { label: "cpu > 80 || memory > 90", detail: "CPU or memory high", type: "text", boost: 10 },
    {
      label: "any(mounts, .usedPercent >= 85)",
      detail: "alert when any mount is over 85% full",
      type: "text",
      boost: 10,
    },
  ];
}

export function createEventHints(): Completion[] {
  return [
    { label: "name", detail: "event name", type: "property" },
    { label: "attributes", detail: "event attributes map", type: "property" },
    { label: 'attributes["healthStatus"]', detail: "healthy or unhealthy (health_status events)", type: "property" },
    { label: 'attributes["exitCode"]', detail: "exit code (die events)", type: "property" },
    { label: 'attributes["signal"]', detail: "signal name (kill events)", type: "property" },
    ...exprOperators,
    { label: '"start"', detail: "container started", type: "string" },
    { label: '"stop"', detail: "container stopped", type: "string" },
    { label: '"die"', detail: "container died", type: "string" },
    { label: '"restart"', detail: "container restarted", type: "string" },
    { label: '"kill"', detail: "container killed by signal", type: "string" },
    { label: '"oom"', detail: "container out of memory", type: "string" },
    { label: '"health_status"', detail: "health check changed", type: "string" },
    { label: 'name == "die"', detail: "match container death", type: "text", boost: 10 },
    { label: 'name == "health_status"', detail: "match health changes", type: "text", boost: 10 },
    {
      label: 'name == "health_status" && attributes["healthStatus"] == "unhealthy"',
      detail: "match unhealthy containers",
      type: "text",
      boost: 10,
    },
    { label: 'name in ["stop", "die"]', detail: "match stop or death", type: "text", boost: 10 },
  ];
}

function createAutocomplete(getHints: () => Completion[]): CompletionSource {
  return (context: CompletionContext) => {
    const word = context.matchBefore(/[\w"=!&|.]+/);
    if (!word && !context.explicit) return null;

    const currentWord = word ? word.text.toLowerCase() : "";
    const hints = getHints();
    const filtered = currentWord ? hints.filter((h) => h.label.toLowerCase().includes(currentWord)) : hints;

    return { from: word ? word.from : context.pos, options: filtered };
  };
}

// Keywords and operator words of the expr language (https://expr-lang.org). Symbol operators
// are tokenized separately below.
const exprKeywords = new Set([
  "contains",
  "startsWith",
  "endsWith",
  "matches",
  "in",
  "not",
  "and",
  "or",
  "any",
  "all",
  "none",
  "one",
  "filter",
  "map",
  "count",
  "len",
  "nil",
]);

const exprBooleans = new Set(["true", "false"]);

/**
 * Minimal tokenizer for expr expressions. Without a language attached, CodeMirror never
 * assigns highlight tags, so the editor renders as flat unstyled text — this is what makes
 * `exprHighlightStyle` below actually do something.
 */
function tokenizeExpr(stream: StringStream): string | null {
  if (stream.eatSpace()) return null;

  const char = stream.peek();
  if (char === undefined) {
    stream.next();
    return null;
  }

  // Strings — both quote styles, with escapes.
  if (char === '"' || char === "'") {
    const quote = stream.next();
    let escaped = false;
    let ch: string | void;
    while ((ch = stream.next()) !== undefined) {
      if (ch === quote && !escaped) break;
      escaped = !escaped && ch === "\\";
    }
    return "string";
  }

  if (stream.match(/^\d+(\.\d+)?/)) return "number";

  // Identifiers, keywords and booleans. A leading dot covers the `.usedPercent` form used
  // inside `any(mounts, ...)` predicates.
  if (stream.match(/^\.?[A-Za-z_][\w]*/)) {
    const word = stream.current().replace(/^\./, "");
    if (exprBooleans.has(word)) return "bool";
    if (exprKeywords.has(word)) return "keyword";
    return "propertyName";
  }

  if (stream.match(/^(==|!=|>=|<=|&&|\|\||[<>!+\-*/%?:])/)) return "operator";

  stream.next();
  return null;
}

export async function createExprEditor(options: ExprEditorOptions) {
  const [
    { EditorView, keymap, placeholder, drawSelection },
    { EditorState, Prec },
    { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap },
    { HighlightStyle, syntaxHighlighting, StreamLanguage },
    { history, historyKeymap, defaultKeymap },
    { tags },
  ] = await Promise.all([
    import("@codemirror/view"),
    import("@codemirror/state"),
    import("@codemirror/autocomplete"),
    import("@codemirror/language"),
    import("@codemirror/commands"),
    import("@lezer/highlight"),
  ]);

  const exprLanguage = StreamLanguage.define({
    name: "expr",
    token: tokenizeExpr,
    languageData: {
      closeBrackets: { brackets: ["(", "[", '"', "'"] },
    },
  });

  const editorTheme = EditorView.theme({
    "&": {
      backgroundColor: "transparent",
      color: "var(--color-base-content)",
      fontSize: "0.875rem",
      width: "100%",
    },
    "&.cm-editor.cm-focused": {
      // The wrapping DaisyUI `.input` already draws the focus ring.
      outline: "none",
    },
    ".cm-scroller": {
      fontFamily: "var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace)",
      lineHeight: "1.6",
    },
    ".cm-content": {
      caretColor: "var(--color-primary)",
      padding: "0.375rem 0",
    },
    ".cm-line": {
      padding: "0",
    },
    ".cm-cursor": {
      borderLeftColor: "var(--color-primary)",
      borderLeftWidth: "2px",
    },
    ".cm-placeholder": {
      color: "color-mix(in oklch, var(--color-base-content) 40%, transparent)",
      fontStyle: "normal",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
      backgroundColor: "color-mix(in oklch, var(--color-primary) 25%, transparent)",
    },
    ".cm-activeLine": {
      // A single-line expression field looks broken with a highlighted "active line".
      backgroundColor: "transparent",
    },
    ".cm-tooltip": {
      backgroundColor: "var(--color-base-200)",
      border: "1px solid color-mix(in oklch, var(--color-base-content) 15%, transparent)",
      borderRadius: "var(--radius-box, 0.5rem)",
      boxShadow: "0 8px 24px rgb(0 0 0 / 0.18)",
      overflow: "hidden",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul": {
      fontFamily: "var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace)",
      fontSize: "0.8125rem",
      maxHeight: "16rem",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li": {
      padding: "0.25rem 0.625rem",
      color: "var(--color-base-content)",
      display: "flex",
      alignItems: "baseline",
      gap: "0.5rem",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]": {
      backgroundColor: "var(--color-primary)",
      color: "var(--color-primary-content)",
    },
    ".cm-completionLabel": {
      flex: "1 1 auto",
    },
    ".cm-completionMatchedText": {
      textDecoration: "none",
      fontWeight: "600",
      color: "var(--color-primary)",
    },
    "li[aria-selected] .cm-completionMatchedText": {
      color: "var(--color-primary-content)",
    },
    ".cm-completionDetail": {
      fontStyle: "normal",
      fontSize: "0.75rem",
      opacity: "0.6",
      flex: "0 0 auto",
    },
  });

  const highlightStyle = HighlightStyle.define([
    { tag: tags.keyword, color: "var(--color-secondary)", fontWeight: "600" },
    { tag: tags.operator, color: "color-mix(in oklch, var(--color-base-content) 70%, transparent)" },
    { tag: tags.string, color: "var(--color-success)" },
    { tag: tags.number, color: "var(--color-warning)" },
    { tag: tags.bool, color: "var(--color-warning)" },
    { tag: tags.propertyName, color: "var(--color-info)" },
    { tag: tags.variableName, color: "var(--color-base-content)" },
  ]);

  const state = EditorState.create({
    doc: options.initialValue,
    extensions: [
      EditorView.lineWrapping,
      exprLanguage,
      history(),
      drawSelection(),
      closeBrackets(),
      placeholder(options.placeholder),
      autocompletion({
        override: [createAutocomplete(options.getHints)],
        activateOnTyping: true,
        icons: false,
      }),
      // Expressions are single-line: swallow Enter unless it is picking a completion,
      // which completionKeymap handles first. Higher precedence than the default keymap's
      // insertNewlineAndIndent.
      Prec.high(
        keymap.of([
          {
            key: "Enter",
            run: () => true,
          },
        ]),
      ),
      keymap.of([...closeBracketsKeymap, ...completionKeymap, ...historyKeymap, ...defaultKeymap]),
      editorTheme,
      syntaxHighlighting(highlightStyle),
      // Pasting a wrapped expression out of the docs would otherwise leave a multi-line
      // document that can never compile. Flatten it instead of rejecting the paste.
      EditorView.domEventHandlers({
        paste(event, view) {
          const text = event.clipboardData?.getData("text/plain");
          if (!text || !/[\r\n]/.test(text)) return false;
          event.preventDefault();
          view.dispatch(view.state.replaceSelection(text.replace(/\s*[\r\n]+\s*/g, " ").trim()));
          return true;
        },
      }),
      EditorView.updateListener.of((update) => {
        if (update.docChanged && options.onChange) {
          options.onChange(update.view.state.doc.toString());
        }
      }),
    ],
  });

  return new EditorView({ state, parent: options.parent });
}
