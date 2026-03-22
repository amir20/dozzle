import { createExprEditor } from "@/composable/exprEditor";

type ExprEditorOptions = Parameters<typeof createExprEditor>[0];

export function useExprEditorField(
  editorRef: Ref<HTMLElement | undefined>,
  options: Omit<ExprEditorOptions, "parent">,
) {
  let editorView: Awaited<ReturnType<typeof createExprEditor>> | undefined;

  onMounted(async () => {
    if (editorRef.value) {
      editorView = await createExprEditor({
        parent: editorRef.value,
        ...options,
      });
    }
  });

  onScopeDispose(() => {
    editorView?.destroy();
  });
}
