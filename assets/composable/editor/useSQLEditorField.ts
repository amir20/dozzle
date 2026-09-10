import { createSQLEditor } from "./sqlEditor";

type SQLEditorOptions = Parameters<typeof createSQLEditor>[0];

/** Mount/teardown wrapper around createSQLEditor. Mirrors useExprEditorField. */
export function useSQLEditorField(editorRef: Ref<HTMLElement | undefined>, options: Omit<SQLEditorOptions, "parent">) {
  let editorView: Awaited<ReturnType<typeof createSQLEditor>> | undefined;
  // CodeMirror loads lazily, so a setValue that lands before it is ready (an example
  // chip clicked on a fast connection) is applied once it mounts.
  let pending: string | undefined;

  onMounted(async () => {
    if (editorRef.value) {
      editorView = await createSQLEditor({ parent: editorRef.value, ...options });
      if (pending !== undefined) {
        const value = pending;
        pending = undefined;
        setValue(value);
      }
    }
  });

  onScopeDispose(() => editorView?.destroy());

  /** Replaces the editor's content, e.g. when an example is picked. */
  function setValue(value: string) {
    if (!editorView) {
      pending = value;
      options.onChange?.(value);
      return;
    }
    editorView.dispatch({
      changes: { from: 0, to: editorView.state.doc.length, insert: value },
      selection: { anchor: value.length },
    });
    editorView.focus();
  }

  /** Inserts text at the cursor, e.g. a column name from the column list. */
  function insertAtCursor(text: string) {
    if (!editorView) return;
    const { from, to } = editorView.state.selection.main;
    editorView.dispatch({
      changes: { from, to, insert: text },
      selection: { anchor: from + text.length },
    });
    editorView.focus();
  }

  return { setValue, insertAtCursor };
}
