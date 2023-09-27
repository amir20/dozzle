type Toast = {
  id: number;
  createdAt: Date;
  message: string;
  type: "success" | "error" | "warning" | "info";
};

const toasts = ref<Toast[]>([]);

const showToast = (message: string, type: Toast["type"]) => {
  toasts.value.push({
    id: Date.now(),
    createdAt: new Date(),
    message,
    type,
  });
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
