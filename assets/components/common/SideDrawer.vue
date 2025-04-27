<template>
  <dialog ref="panel" class="modal-right modal items-start outline-hidden backdrop:bg-none">
    <div class="modal-box" :width="width">
      <div class="pt-safe relative">
        <form method="dialog" class="absolute right-0">
          <button v-if="isMobile">
            <mdi:close />
          </button>
          <button v-else class="swap hover:swap-active outline-hidden">
            <mdi:keyboard-esc class="swap-off" />
            <mdi:close class="swap-on" />
          </button>
        </form>
        <slot v-if="open"></slot>
      </div>
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
<style scoped>
@reference "@/main.css";

.modal-right :where(.modal-box) {
  @apply bg-base-100 fixed right-0 h-lvh max-h-screen translate-x-24 scale-100 rounded-none shadow-none;

  &[width="md"] {
    @apply max-w-3xl;
  }

  &[width="lg"] {
    @apply max-w-5xl;
  }
}

.modal-right[open] .modal-box {
  @apply translate-x-0;
}
</style>
