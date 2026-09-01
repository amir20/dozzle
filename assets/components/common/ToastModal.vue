<template>
  <div class="toast toast-end max-md:toast-center max-md:toast-bottom whitespace-normal max-md:w-full max-md:px-2">
    <div
      class="alert flex max-w-xl flex-col items-stretch gap-2 shadow-sm max-md:w-full max-md:rounded-lg"
      v-for="{ toast, options: { timed } } in toasts"
      :key="toast.id"
      :class="{
        'alert-error': toast.type === 'error',
        'alert-info': toast.type === 'info',
        'alert-warning': toast.type === 'warning',
      }"
    >
      <div class="flex w-full items-start gap-3">
        <carbon:information class="size-5 shrink-0 stroke-current" v-if="toast.type === 'info'" />
        <carbon:warning class="size-5 shrink-0 stroke-current" v-else-if="toast.type === 'error'" />
        <carbon:warning class="size-5 shrink-0 stroke-current" v-else-if="toast.type === 'warning'" />
        <div class="min-w-0 grow">
          <h3 class="text-lg font-bold max-md:text-base" v-if="toast.title">{{ toast.title }}</h3>
          <div v-html="toast.message" class="max-md:text-sm [&>a]:underline"></div>
          <div class="mt-2 flex items-center gap-2" v-if="toast.progress !== undefined">
            <progress class="progress progress-primary h-1.5 grow" :value="toast.progress" max="100"></progress>
            <span class="text-xs tabular-nums opacity-70">{{ Math.round(toast.progress) }}%</span>
          </div>
        </div>
        <button class="btn btn-circle btn-xs shrink-0" @click="removeToast(toast.id)">
          <mdi:close />
        </button>
      </div>

      <!-- Actions sit under the message so a long notice keeps its full width
           instead of being squeezed by the buttons beside it. -->
      <div class="flex w-full justify-end gap-1" v-if="timed || toast.action || toast.secondaryAction">
        <TimedButton
          v-if="timed"
          class="btn-primary btn-sm"
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
            class="btn btn-ghost btn-sm"
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
            class="btn btn-primary btn-sm"
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
  </div>
</template>
<script lang="ts" setup>
const { toasts, removeToast } = useToast();
</script>
