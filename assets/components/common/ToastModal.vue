<template>
  <!--
    Toasts sit on top of the log stream, so they read as a panel in the same
    family as the dropdowns (neutral surface, hairline border, rounded-box)
    rather than a saturated block of color. Severity shows up as a rail and a
    tinted icon instead of tinting the whole notice, which keeps the text at
    full contrast.
  -->
  <TransitionGroup
    tag="div"
    name="toast"
    class="toast toast-end max-md:toast-center max-md:toast-bottom whitespace-normal max-md:w-full max-md:px-2"
  >
    <div
      class="rounded-box border-base-content/15 bg-base-200/95 relative flex w-96 max-w-full flex-col gap-2.5 overflow-hidden border p-3 shadow-lg backdrop-blur max-md:w-full"
      v-for="{ toast, options: { timed } } in toasts"
      :key="toast.id"
    >
      <span class="absolute inset-y-0 left-0 w-1" :class="rail[toast.type]"></span>

      <div class="flex w-full items-start gap-2.5">
        <div class="shrink-0 rounded-full p-1.5" :class="chip[toast.type]">
          <mdi:information-outline class="size-4" v-if="toast.type === 'info'" />
          <mdi:alert-circle-outline class="size-4" v-else-if="toast.type === 'error'" />
          <mdi:alert-outline class="size-4" v-else />
        </div>
        <div class="min-w-0 grow">
          <h3 class="text-sm leading-5 font-semibold" v-if="toast.title">{{ toast.title }}</h3>
          <div
            v-html="toast.message"
            class="text-base-content/70 text-xs leading-relaxed [&>a]:underline"
            :class="{ 'mt-1': toast.title }"
          ></div>
          <div class="mt-2 flex items-center gap-2" v-if="toast.progress !== undefined">
            <progress class="progress progress-primary h-1 grow" :value="toast.progress" max="100"></progress>
            <span class="text-base-content/50 font-mono text-[0.65rem] tabular-nums">
              {{ Math.round(toast.progress) }}%
            </span>
          </div>
        </div>
        <button class="btn btn-circle btn-ghost btn-xs shrink-0" @click="removeToast(toast.id)">
          <mdi:close class="size-3.5" />
        </button>
      </div>

      <!-- Actions sit under the message so a long notice keeps its full width
           instead of being squeezed by the buttons beside it. -->
      <div class="flex w-full justify-end gap-1" v-if="timed || toast.action || toast.secondaryAction">
        <TimedButton
          v-if="timed"
          class="btn-primary btn-xs"
          :duration="timed"
          @finished="
            removeToast(toast.id);
            toast.action?.handler();
          "
          @cancelled="removeToast(toast.id)"
        >
          {{ toast.action?.label }}
        </TimedButton>
        <template v-else>
          <button
            class="btn btn-ghost btn-xs"
            v-if="toast.secondaryAction"
            @click="
              toast.secondaryAction.handler();
              removeToast(toast.id);
            "
          >
            {{ toast.secondaryAction.label }}
          </button>
          <!-- Acting on a notice answers it, so the toast goes away either way. -->
          <button
            class="btn btn-primary btn-xs"
            v-if="toast.action"
            @click="
              toast.action.handler();
              removeToast(toast.id);
            "
          >
            {{ toast.action.label }}
          </button>
        </template>
      </div>
    </div>
  </TransitionGroup>
</template>
<script lang="ts" setup>
const { toasts, removeToast } = useToast();

const rail = {
  error: "bg-error",
  warning: "bg-warning",
  info: "bg-info",
} as const;

const chip = {
  error: "bg-error/15 text-error",
  warning: "bg-warning/15 text-warning",
  info: "bg-info/15 text-info",
} as const;
</script>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition:
    opacity 200ms ease,
    transform 200ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(1rem) scale(0.98);
}

/* Stacked toasts slide into the gap a dismissed one leaves behind. */
.toast-move {
  transition: transform 200ms ease;
}

@media (width < 48rem) {
  .toast-enter-from,
  .toast-leave-to {
    transform: translateY(1rem) scale(0.98);
  }
}

@media (prefers-reduced-motion: reduce) {
  .toast-enter-active,
  .toast-leave-active,
  .toast-move {
    transition: opacity 200ms ease;
  }
  .toast-enter-from,
  .toast-leave-to {
    transform: none;
  }
}
</style>
