<template>
  <!-- Near-fullscreen settings popup. Same frosted backdrop as the container
       search popup. The box is fixed-size with an even margin around the screen
       and does not scroll; only the body inside scrolls when it needs to. -->
  <dialog ref="dialog" class="modal bg-base-300/50! w-screen backdrop-blur-md transition-none!" @close="closeSettings">
    <div
      class="modal-box border-base-content/15 flex h-[calc(100%-6rem)] w-[calc(100%-6rem)] max-w-none flex-col overflow-hidden! rounded-2xl border p-0 shadow-2xl"
    >
      <header class="border-base-content/10 flex shrink-0 items-center justify-between gap-4 border-b px-6 py-4">
        <div class="flex items-center gap-3">
          <mdi:cog class="text-base-content/70 size-6" />
          <h2 class="text-xl font-semibold tracking-tight">{{ $t("title.settings") }}</h2>
          <span class="status-pill status-pill-neutral font-mono">{{ config.version }}</span>
        </div>
        <div class="flex items-center gap-2">
          <kbd class="kbd kbd-sm hidden sm:inline-flex">esc</kbd>
          <form method="dialog">
            <button class="btn btn-sm btn-circle btn-ghost" :aria-label="$t('toolbar.close')">
              <mdi:close class="size-5" />
            </button>
          </form>
        </div>
      </header>

      <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-6 py-6">
        <SettingsPanel v-if="open" columns />
      </div>
    </div>

    <form method="dialog" class="modal-backdrop">
      <button>close</button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import SettingsPanel from "@/components/Settings/SettingsPanel.vue";
import { useSettingsModal } from "@/composable/settingsModal";

const { open, closeSettings } = useSettingsModal();
const dialog = ref<HTMLDialogElement>();

// Hide the page scrollbar while open so the fixed near-fullscreen box keeps an
// even margin on all sides (otherwise a scrollbar on the page behind eats into
// the right margin). The shifted content sits behind the frosted backdrop.
watch(open, (visible) => {
  if (visible) {
    // Remove the page scrollbar first so the dialog lays out against the full
    // viewport width; otherwise showModal() sizes it minus the scrollbar and
    // the right margin ends up larger than the left.
    document.documentElement.style.overflow = "hidden";
    dialog.value?.showModal();
  } else {
    dialog.value?.close();
    document.documentElement.style.overflow = "";
  }
});

onUnmounted(() => {
  document.documentElement.style.overflow = "";
});
</script>
