<template>
  <dialog ref="panel" class="modal-right modal items-start outline-hidden backdrop:bg-none">
    <div class="modal-box" :width="maximized ? 'full' : width">
      <div class="pt-safe relative">
        <div class="absolute right-0 z-10 flex items-center gap-3">
          <button
            v-if="!isMobile"
            class="icon-btn hover:text-base-content/60 outline-hidden"
            type="button"
            :title="maximized ? $t('drawer.restore') : $t('drawer.maximize')"
            :aria-label="maximized ? $t('drawer.restore') : $t('drawer.maximize')"
            @click="maximized = !maximized"
          >
            <mdi:arrow-collapse v-if="maximized" />
            <mdi:arrow-expand v-else />
          </button>
          <button v-if="isMobile" class="icon-btn" type="button" @click="requestClose">
            <mdi:close />
          </button>
          <button v-else class="icon-btn swap hover:swap-active outline-hidden" type="button" @click="requestClose">
            <mdi:keyboard-esc class="swap-off" />
            <mdi:close class="swap-on" />
          </button>
        </div>
        <slot v-if="open" :close="requestClose"></slot>
      </div>
    </div>
    <button class="modal-backdrop cursor-pointer" type="button" @click="requestClose">close</button>
  </dialog>
</template>
<script setup lang="ts">
import { type DrawerWidth, type DrawerCloseGuard } from "@/composable/drawer";
const panel = useTemplateRef<HTMLDialogElement>("panel");

const open = ref(false);
const maximized = ref(false);
const { width, closeGuard } = defineProps<{
  width: DrawerWidth;
  /** Set by the drawer's occupant to veto a close, e.g. to confirm discarding a form. */
  closeGuard?: DrawerCloseGuard | null;
}>();

function close() {
  panel.value?.close();
}

/** Every close path goes through the guard: Esc, the backdrop, the close button, the slot. */
function requestClose() {
  if (closeGuard && !closeGuard()) return;
  close();
}

defineExpose({
  open: () => {
    open.value = true;
    maximized.value = false;
    panel.value?.showModal();
  },
  close,
});

// Esc closes a native dialog without going through any of our handlers.
useEventListener(panel, "cancel", (event: Event) => {
  if (closeGuard && !closeGuard()) event.preventDefault();
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
