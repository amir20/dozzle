import { createExprEditor } from "@/composable/exprEditor";

type ExprEditorOptions = Parameters<typeof createExprEditor>[0];

export function useExprEditorField(
  editorRef: Ref<HTMLElement | undefined>,
  options: Omit<ExprEditorOptions, "parent">,
) {
  let editorView: Awaited<ReturnType<typeof createExprEditor>> | undefined;
  // The editor loads CodeMirror lazily, so a setValue that lands before it is ready
  // (an example chip clicked on a fast connection) is applied once it mounts.
  let pending: string | undefined;

  onMounted(async () => {
    if (editorRef.value) {
      editorView = await createExprEditor({
        parent: editorRef.value,
        ...options,
      });
      if (pending !== undefined) {
        const value = pending;
        pending = undefined;
        setValue(value);
      }
    }
  });

  onScopeDispose(() => {
    editorView?.destroy();
  });

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

  return { setValue };
}
