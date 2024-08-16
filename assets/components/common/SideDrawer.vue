<template>
  <dialog ref="panel" class="modal-right modal items-start outline-none backdrop:bg-none">
    <div class="modal-box">
      <slot></slot>
    </div>
  </dialog>
</template>
<script setup lang="ts">
const panel = ref<HTMLDialogElement>();
const open = ref(false);
onKeyStroke("o", (e) => {
  if ((e.ctrlKey || e.metaKey) && !e.shiftKey) {
    panel.value?.showModal();
    e.preventDefault();
  }
});
watch(open, () => {
  if (open.value) {
    panel.value?.showModal();
  } else {
    panel.value?.close();
  }
});

defineExpose({ open: () => (open.value = true) });
</script>
<style scoped lang="postcss">
.modal-right :where(.modal-box) {
  @apply fixed right-0 h-lvh max-h-screen max-w-2xl translate-x-24 scale-100 rounded-none bg-base-lighter shadow-none;
}

.modal-right[open] .modal-box {
  @apply translate-x-0;
}
</style>
