<template>
  <dialog ref="dialog" class="modal" @close="$emit('close')" @cancel="$emit('cancel', $event)">
    <div class="modal-box flex max-h-[90vh] w-full max-w-215 overflow-hidden p-0 max-md:flex-col md:h-160">
      <!-- Rail: where you are and how much is left. On a phone it collapses to a bar. -->
      <aside class="bg-base-200/60 border-base-content/10 w-55 shrink-0 border-r p-5 max-md:hidden">
        <div class="text-base font-semibold">{{ title }}</div>
        <ol class="mt-5 flex flex-col gap-1">
          <li
            v-for="(step, i) in steps"
            :key="step.id"
            class="flex items-start gap-2.5 rounded-md px-2 py-1.5"
            :class="{ 'bg-base-300': step.state === 'current' }"
            :aria-current="step.state === 'current' ? 'step' : undefined"
          >
            <span
              class="mt-px flex size-5 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
              :class="chipClass[step.state]"
            >
              <mdi:check v-if="step.state === 'done'" class="size-3.5" />
              <template v-else>{{ i + 1 }}</template>
            </span>
            <span class="min-w-0">
              <span class="block text-sm" :class="step.state === 'current' ? 'font-semibold' : 'text-base-content/70'">
                {{ step.label }}
              </span>
              <span v-if="step.note" class="text-base-content/40 block text-xs">{{ step.note }}</span>
            </span>
          </li>
        </ol>
      </aside>

      <div class="flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="flex gap-1 px-4 pt-4 md:hidden" aria-hidden="true">
          <span
            v-for="step in steps"
            :key="step.id"
            class="h-0.75 flex-1 rounded-full transition-colors"
            :class="barClass[step.state]"
          ></span>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto p-8 max-md:p-5">
          <slot></slot>
        </div>

        <div class="border-base-content/10 bg-base-100 flex items-center gap-2 border-t px-8 py-4 max-md:px-5">
          <slot name="footer-start"></slot>
          <div class="ml-auto flex items-center gap-2">
            <slot name="footer-end"></slot>
          </div>
        </div>
      </div>
    </div>
  </dialog>
</template>

<script lang="ts" setup>
// The frame shared by every multi-step modal: a numbered rail, a scrolling body and
// a sticky footer. The parent owns what the steps are and what the buttons do.
export type StepModalState = "done" | "current" | "todo" | "skipped";
export interface StepModalStep {
  id: string;
  label: string;
  note?: string;
  state: StepModalState;
}

defineProps<{ title: string; steps: StepModalStep[] }>();
defineEmits<{ close: []; cancel: [event: Event] }>();

const dialog = ref<HTMLDialogElement>();

const chipClass: Record<StepModalState, string> = {
  done: "bg-success text-success-content",
  current: "bg-primary text-primary-content",
  skipped: "border-base-content/30 text-base-content/40 border border-dashed",
  todo: "bg-base-content/10 text-base-content/60",
};

const barClass: Record<StepModalState, string> = {
  done: "bg-primary/45",
  skipped: "bg-primary/45",
  current: "bg-primary",
  todo: "bg-base-content/15",
};

defineExpose({
  open: () => {
    if (!dialog.value?.open) dialog.value?.showModal();
  },
  close: () => dialog.value?.close(),
  isOpen: () => !!dialog.value?.open,
});
</script>
