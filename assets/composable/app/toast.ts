type Toast = {
  id: string;
  createdAt: Date;
  title?: string;
  message: string;
  type: "error" | "warning" | "info";
  action?: {
    label: string;
    handler: () => void;
  };
  // Rendered next to the primary action as a quieter choice, e.g. silencing a
  // recurring notice rather than just closing it.
  secondaryAction?: {
    label: string;
    handler: () => void;
  };
  // Percentage for a determinate progress bar. Omit for work that cannot
  // report how far along it is.
  progress?: number;
};

type ToastOptions = {
  expire?: number;
  once?: boolean;
  timed?: number;
};

const toasts = ref<
  {
    toast: Toast;
    options: ToastOptions;
  }[]
>([]);

const showToast = (
  toast: Omit<Toast, "id" | "createdAt"> & { id?: string },
  { expire = -1, once = false, timed }: ToastOptions = { expire: -1, once: false },
) => {
  if (once && !toast.id) {
    throw new Error("Toast id is required when once is true");
  }
  if (once && toasts.value.some((t) => t.toast.id === toast.id)) {
    return;
  }

  const toastWithId = {
    id: Date.now().toString(),
    ...toast,
    createdAt: new Date(),
  };
  toasts.value.push({
    toast: toastWithId,
    options: { expire, once, timed },
  });

  if (expire > 0) {
    setTimeout(() => {
      removeToast(toastWithId.id);
    }, expire);
  }
};

// Patches a toast that is already on screen, so long-running work can report
// progress without the notice being torn down and rebuilt on every tick.
const updateToast = (id: Toast["id"], patch: Partial<Omit<Toast, "id" | "createdAt">>) => {
  const existing = toasts.value.find((instance) => instance.toast.id === id);
  if (existing) {
    Object.assign(existing.toast, patch);
  }
};

const removeToast = (id: Toast["id"]) => {
  toasts.value = toasts.value.filter((instance) => instance.toast.id !== id);
};

export const useToast = () => {
  return {
    toasts,
    showToast,
    updateToast,
    removeToast,
  };
};
