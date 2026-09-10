import type { Completion } from "@codemirror/autocomplete";
import type { DecorationSet, EditorView as EditorViewType, ViewUpdate } from "@codemirror/view";

export interface TemplateEditorOptions {
  parent: HTMLElement;
  initialValue: string;
  placeholder?: string;
  onChange?: (value: string) => void;
}

/**
 * Fields a webhook template can read, as Go template paths.
 *
 * `scope` says which alert types populate the value: `Log`, `Stat` and `Event` are pointers on
 * the notification and are nil for the other two kinds, so `{{ .Log.Level }}` on a metric alert
 * fails at send time rather than rendering empty.
 */
export interface TemplateVariable {
  path: string;
  detail: string;
  scope: "all" | "log" | "metric" | "event";
}

export const TEMPLATE_VARIABLES: TemplateVariable[] = [
  { path: ".Detail", detail: "human-readable summary of what fired", scope: "all" },
  { path: ".Type", detail: "log, metric or event", scope: "all" },
  { path: ".Timestamp", detail: "when the alert fired", scope: "all" },
  { path: ".Container.Name", detail: "container name", scope: "all" },
  { path: ".Container.ID", detail: "container id", scope: "all" },
  { path: ".Container.Image", detail: "container image", scope: "all" },
  { path: ".Container.State", detail: "running, exited, etc.", scope: "all" },
  { path: ".Container.Health", detail: "healthy, unhealthy, none", scope: "all" },
  { path: ".Container.HostName", detail: "docker host name", scope: "all" },
  { path: ".Container.HostID", detail: "docker host id", scope: "all" },
  { path: ".Subscription.Name", detail: "name of the alert that fired", scope: "all" },
  { path: ".Log.Message", detail: "matched log line", scope: "log" },
  { path: ".Log.Level", detail: "log level", scope: "log" },
  { path: ".Log.Stream", detail: "stdout or stderr", scope: "log" },
  { path: ".Stat.CPUPercent", detail: "CPU usage percent", scope: "metric" },
  { path: ".Stat.MemoryPercent", detail: "memory usage percent", scope: "metric" },
  { path: ".Stat.MemoryUsage", detail: "memory usage in bytes", scope: "metric" },
  { path: ".Event.Name", detail: "start, stop, die, health_status…", scope: "event" },
];

export type PayloadMode = "json" | "raw" | "unbalanced" | "empty";

/**
 * Classifies how the backend will treat this template.
 *
 * `executeJSONTemplate` unmarshals the raw template text first, so a template that is valid JSON
 * with its `{{ }}` actions still in place gets each string value rendered and re-marshalled
 * (values are JSON-escaped for you). Anything else falls back to running the whole text through
 * text/template, where escaping is your problem. Both are supported, so this reports which one
 * you are in rather than calling either wrong.
 */
export function payloadMode(template: string): PayloadMode {
  const text = template.trim();
  if (!text) return "empty";

  // Cheap balance check first: an unclosed action is a genuine error in either mode.
  let depth = 0;
  for (let i = 0; i < text.length - 1; i++) {
    if (text[i] === "{" && text[i + 1] === "{") {
      depth++;
      i++;
    } else if (text[i] === "}" && text[i + 1] === "}" && depth > 0) {
      depth--;
      i++;
    }
  }
  if (depth !== 0) return "unbalanced";

  try {
    JSON.parse(text);
    return "json";
  } catch {
    return "raw";
  }
}

function templateCompletions(): Completion[] {
  return TEMPLATE_VARIABLES.map((v) => ({
    label: `{{ ${v.path} }}`,
    detail: v.detail,
    type: "property",
  }));
}

export async function createTemplateEditor(options: TemplateEditorOptions) {
  const [
    { EditorView, keymap, placeholder, Decoration, MatchDecorator, ViewPlugin },
    { EditorState },
    { json },
    { HighlightStyle, syntaxHighlighting },
    { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap },
    { history, historyKeymap, defaultKeymap, indentWithTab },
    { tags },
  ] = await Promise.all([
    import("@codemirror/view"),
    import("@codemirror/state"),
    import("@codemirror/lang-json"),
    import("@codemirror/language"),
    import("@codemirror/autocomplete"),
    import("@codemirror/commands"),
    import("@lezer/highlight"),
  ]);

  // The JSON grammar sees `{{ .Container.Name }}` as ordinary string content, so the part that
  // actually does something renders like the text around it. Decorate the actions on top.
  const actionMatcher = new MatchDecorator({
    regexp: /\{\{[^}]*\}\}/g,
    decoration: Decoration.mark({ class: "cm-template-action" }),
  });

  const templateActions = ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;
      constructor(view: EditorViewType) {
        this.decorations = actionMatcher.createDeco(view);
      }
      update(update: ViewUpdate) {
        this.decorations = actionMatcher.updateDeco(update, this.decorations);
      }
    },
    { decorations: (v) => v.decorations },
  );

  const editorTheme = EditorView.theme({
    "&": {
      backgroundColor: "var(--color-base-100)",
      color: "var(--color-base-content)",
      fontSize: "0.875rem",
    },
    "&.cm-editor.cm-focused": {
      // The wrapping container already draws the focus ring.
      outline: "none",
    },
    ".cm-content": {
      caretColor: "var(--color-primary)",
      fontFamily: "var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace)",
      padding: "0.5rem 0",
    },
    ".cm-cursor": {
      borderLeftColor: "var(--color-primary)",
      borderLeftWidth: "2px",
    },
    ".cm-placeholder": {
      color: "color-mix(in oklch, var(--color-base-content) 40%, transparent)",
      fontStyle: "normal",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
      backgroundColor: "color-mix(in oklch, var(--color-primary) 25%, transparent)",
    },
    ".cm-activeLine": {
      backgroundColor: "color-mix(in oklch, var(--color-base-200) 50%, transparent)",
    },
    ".cm-gutters": {
      backgroundColor: "transparent",
      color: "color-mix(in oklch, var(--color-base-content) 40%, transparent)",
      border: "none",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "transparent",
      color: "var(--color-base-content)",
    },
    ".cm-template-action": {
      color: "var(--color-secondary)",
      backgroundColor: "color-mix(in oklch, var(--color-secondary) 12%, transparent)",
      borderRadius: "3px",
      fontWeight: "600",
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
    ".cm-completionLabel": { flex: "1 1 auto" },
    ".cm-completionMatchedText": {
      textDecoration: "none",
      fontWeight: "600",
      color: "var(--color-primary)",
    },
    "li[aria-selected] .cm-completionMatchedText": { color: "var(--color-primary-content)" },
    ".cm-completionDetail": {
      fontStyle: "normal",
      fontSize: "0.75rem",
      opacity: "0.6",
      flex: "0 0 auto",
    },
  });

  const highlightStyle = HighlightStyle.define([
    { tag: tags.propertyName, color: "var(--color-info)" },
    { tag: tags.string, color: "var(--color-success)" },
    { tag: tags.number, color: "var(--color-warning)" },
    { tag: tags.bool, color: "var(--color-warning)" },
    { tag: tags.null, color: "var(--color-secondary)" },
    { tag: tags.punctuation, color: "color-mix(in oklch, var(--color-base-content) 70%, transparent)" },
  ]);

  const state = EditorState.create({
    doc: options.initialValue,
    extensions: [
      EditorView.lineWrapping,
      json(),
      templateActions,
      history(),
      closeBrackets(),
      ...(options.placeholder ? [placeholder(options.placeholder)] : []),
      autocompletion({
        override: [
          (context) => {
            // Offer the notification fields as soon as an action is opened, and on demand.
            const word = context.matchBefore(/[{.\w]+/);
            if (!word && !context.explicit) return null;
            return { from: word ? word.from : context.pos, options: templateCompletions() };
          },
        ],
        icons: false,
      }),
      keymap.of([...closeBracketsKeymap, ...completionKeymap, ...historyKeymap, ...defaultKeymap, indentWithTab]),
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
