type Toast = {
  id: string;
  createdAt: Date;
  title?: string;
  message: string;
  type: "success" | "error" | "warning" | "info";
  action?: {
    label: string;
    handler: () => void;
  };
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

const removeToast = (id: Toast["id"]) => {
  toasts.value = toasts.value.filter((instance) => instance.toast.id !== id);
};

export const useToast = () => {
  return {
    toasts,
    showToast,
    removeToast,
  };
};
