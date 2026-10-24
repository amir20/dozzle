<script lang="ts" setup>
import { withBase } from "vitepress";
import { ref } from "vue";

// The hero already shows a live demo, so the tour is opt-in: a thumbnail next to the
// calls to action that opens the video with sound. Nothing downloads until it's opened.
const dialog = ref<HTMLDialogElement>();
const video = ref<HTMLVideoElement>();

function open() {
  dialog.value?.showModal();
  video.value?.play().catch(() => {});
}

function onClose() {
  if (!video.value) return;
  video.value.pause();
  video.value.currentTime = 0;
}
</script>

<template>
  <button
    type="button"
    class="group mx-auto mt-8 flex w-fit cursor-pointer items-center gap-3 rounded-xl! border border-solid border-(--vp-c-divider) bg-(--vp-c-bg-soft)! p-1.5! pr-5! text-left transition-colors hover:border-(--vp-c-brand) min-[960px]:mx-0"
    aria-haspopup="dialog"
    @click="open"
  >
    <span class="relative block h-14 w-24 shrink-0 overflow-hidden rounded-lg">
      <img
        :src="withBase('/video/dozzle-tour.jpg')"
        alt=""
        class="size-full object-cover transition-transform duration-300 group-hover:scale-105"
      />
      <span class="absolute inset-0 flex items-center justify-center bg-black/30">
        <span class="flex size-7 items-center justify-center rounded-full bg-white/90 text-black shadow">
          <Icon icon="mdi:play" class="ml-0.5 size-5" />
        </span>
      </span>
    </span>
    <span class="flex flex-col">
      <span class="text-sm font-medium text-(--vp-c-text-1) group-hover:text-(--vp-c-brand)">Watch the tour</span>
      <span class="font-mono text-xs text-(--vp-c-text-3)">0:33 · logs, SQL, chat, updates</span>
    </span>
  </button>

  <dialog
    ref="dialog"
    class="m-auto w-[min(92vw,72rem)] overflow-visible border-0 bg-transparent p-0 backdrop:bg-black/80 backdrop:backdrop-blur-sm"
    aria-label="Dozzle tour video"
    @close="onClose"
    @click.self="dialog?.close()"
  >
    <video
      ref="video"
      class="block aspect-video w-full rounded-xl bg-black shadow-2xl"
      :poster="withBase('/video/dozzle-tour.jpg')"
      controls
      playsinline
      preload="none"
    >
      <source :src="withBase('/video/dozzle-tour.mp4')" type="video/mp4" />
    </video>
    <button
      type="button"
      class="absolute -top-11 right-0 flex cursor-pointer items-center gap-1 rounded-md! px-2! py-1! text-sm text-white/70 transition-colors hover:text-white"
      aria-label="Close video"
      @click="dialog?.close()"
    >
      <Icon icon="mdi:close" class="size-5" /> Close
    </button>
  </dialog>
</template>
