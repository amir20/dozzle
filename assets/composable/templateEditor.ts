export interface TemplateEditorOptions {
  parent: HTMLElement;
  initialValue: string;
  onChange?: (value: string) => void;
}

export async function createTemplateEditor(options: TemplateEditorOptions) {
  const [{ EditorView }, { EditorState }, { json }, { HighlightStyle, syntaxHighlighting }, { tags }] =
    await Promise.all([
      import("@codemirror/view"),
      import("@codemirror/state"),
      import("@codemirror/lang-json"),
      import("@codemirror/language"),
      import("@lezer/highlight"),
    ]);

  const editorTheme = EditorView.theme({
    "&": {
      backgroundColor: "var(--color-base-100)",
      color: "var(--color-base-content)",
      fontSize: "0.875rem",
    },
    ".cm-content": {
      caretColor: "var(--color-primary)",
      fontFamily: "ui-monospace, monospace",
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
    { tag: tags.propertyName, color: "var(--color-info)" },
    { tag: tags.string, color: "var(--color-success)" },
    { tag: tags.number, color: "var(--color-warning)" },
    { tag: tags.bool, color: "var(--color-warning)" },
    { tag: tags.null, color: "var(--color-secondary)" },
    { tag: tags.punctuation, color: "var(--color-base-content)" },
  ]);

  const state = EditorState.create({
    doc: options.initialValue,
    extensions: [
      EditorView.lineWrapping,
      json(),
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
