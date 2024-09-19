<template>
  <dialog ref="panel" class="modal-right modal items-start outline-none backdrop:bg-none">
    <div class="modal-box" :width="width">
      <form method="dialog">
        <button class="swap swap-rotate absolute right-4 top-4 outline-none hover:swap-active">
          <mdi:keyboard-esc class="swap-off" />
          <mdi:close class="swap-on" />
        </button>
      </form>
      <slot v-if="open"></slot>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
</template>
<script setup lang="ts">
import { type DrawerWidth } from "@/composable/drawer";
const panel = useTemplateRef<HTMLDialogElement>("panel");

const open = ref(false);
const { width } = defineProps<{
  width: DrawerWidth;
}>();

defineExpose({
  open: () => {
    open.value = true;
    panel.value?.showModal();
  },
});

useEventListener(panel, "close", () => (open.value = false));
</script>
<style scoped lang="postcss">
.modal-right :where(.modal-box) {
  @apply fixed right-0 h-lvh max-h-screen translate-x-24 scale-100 rounded-none bg-base-lighter shadow-none;

  &[size="md"] {
    @apply max-w-3xl;
  }

  &[size="lg"] {
    @apply max-w-5xl;
  }
}

.modal-right[open] .modal-box {
  @apply translate-x-0;
}
</style>
