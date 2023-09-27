type Toast = {
  id: number;
  createdAt: Date;
  title?: string;
  message: string;
  type: "success" | "error" | "warning" | "info";
};

const toasts = ref<Toast[]>([]);

const showToast = (toast: Omit<Toast, "id" | "createdAt">, expire = -1) => {
  toasts.value.push({
    id: Date.now(),
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
