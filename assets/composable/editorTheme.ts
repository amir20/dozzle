import type { EditorView as EditorViewType } from "@codemirror/view";
import type { HighlightStyle as HighlightStyleType } from "@codemirror/language";
import type { tags as tagsType } from "@lezer/highlight";

/**
 * The look shared by every CodeMirror field in the app (expression inputs, the SQL
 * console). CodeMirror is loaded lazily, so the constructors are passed in rather than
 * imported here — importing them statically would pull the editor into the entry chunk.
 */
export function createEditorTheme(EditorView: typeof EditorViewType, { multiline = false } = {}) {
  return EditorView.theme({
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
      // A single-line expression field looks broken with a highlighted "active line";
      // a multi-line query is easier to follow with one.
      backgroundColor: multiline ? "color-mix(in oklch, var(--color-base-content) 4%, transparent)" : "transparent",
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
}

export function createHighlightStyle(HighlightStyle: typeof HighlightStyleType, tags: typeof tagsType) {
  return HighlightStyle.define([
    { tag: tags.keyword, color: "var(--color-secondary)", fontWeight: "600" },
    { tag: tags.operator, color: "color-mix(in oklch, var(--color-base-content) 70%, transparent)" },
    { tag: tags.string, color: "var(--color-success)" },
    { tag: tags.number, color: "var(--color-warning)" },
    { tag: tags.bool, color: "var(--color-warning)" },
    { tag: tags.propertyName, color: "var(--color-info)" },
    { tag: tags.variableName, color: "var(--color-base-content)" },
    { tag: tags.comment, color: "color-mix(in oklch, var(--color-base-content) 45%, transparent)" },
    { tag: tags.function(tags.variableName), color: "var(--color-secondary)" },
  ]);
}
