<template>
  <dialog ref="panel" class="modal-right modal items-start outline-hidden backdrop:bg-none">
    <div class="modal-box" :width="maximized ? 'full' : width">
      <div class="pt-safe relative">
        <div class="absolute right-0 flex items-center gap-3">
          <button
            v-if="!isMobile"
            class="hover:text-base-content/60 cursor-pointer outline-hidden"
            type="button"
            :title="maximized ? $t('drawer.restore') : $t('drawer.maximize')"
            :aria-label="maximized ? $t('drawer.restore') : $t('drawer.maximize')"
            @click="maximized = !maximized"
          >
            <mdi:arrow-collapse v-if="maximized" />
            <mdi:arrow-expand v-else />
          </button>
          <form method="dialog">
            <button v-if="isMobile" class="cursor-pointer">
              <mdi:close />
            </button>
            <button v-else class="swap hover:swap-active cursor-pointer outline-hidden">
              <mdi:keyboard-esc class="swap-off" />
              <mdi:close class="swap-on" />
            </button>
          </form>
        </div>
        <slot v-if="open" :close="close"></slot>
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
const maximized = ref(false);
const { width } = defineProps<{
  width: DrawerWidth;
}>();

function close() {
  panel.value?.close();
}

defineExpose({
  open: () => {
    open.value = true;
    maximized.value = false;
    panel.value?.showModal();
  },
  close,
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

  &[width="full"] {
    @apply w-full max-w-full;
  }
}

.modal-right[open] .modal-box {
  @apply translate-x-0;
}
</style>
