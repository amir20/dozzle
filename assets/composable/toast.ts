type Toast = {
  id: string;
  createdAt: Date;
  title?: string;
  message: string;
  type: "success" | "error" | "warning" | "info";
};

type ToastOptions = {
  expire?: number;
  once?: boolean;
};

const toasts = ref<Toast[]>([]);

const showToast = (
  toast: Omit<Toast, "id" | "createdAt"> & { id?: string },
  { expire = -1, once = false }: ToastOptions = { expire: -1, once: false },
) => {
  if (once && toasts.value.some((t) => t.id === toast.id)) {
    return;
  }
  toasts.value.push({
    id: Date.now().toString(),
    createdAt: new Date(),
    ...toast,
  });
  if (expire > 0) {
    setTimeout(() => {
      removeToast(toasts.value[0].id);
    }, expire);
  }
};

const removeToast = (id: Toast["id"]) => {
  toasts.value = toasts.value.filter((toast) => toast.id !== id);
};

export const useToast = () => {
  return {
    toasts,
    showToast,
    removeToast,
  };
};
