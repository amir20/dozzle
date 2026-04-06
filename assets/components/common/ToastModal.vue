<template>
  <div class="toast toast-end max-md:toast-center max-md:toast-bottom whitespace-normal max-md:w-full max-md:px-2">
    <div
      class="alert max-w-xl shadow-sm max-md:w-full max-md:rounded-lg"
      v-for="{ toast, options: { timed } } in toasts"
      :key="toast.id"
      :class="{
        'alert-error': toast.type === 'error',
        'alert-info': toast.type === 'info',
        'alert-warning': toast.type === 'warning',
      }"
    >
      <carbon:information class="size-5 shrink-0 stroke-current" v-if="toast.type === 'info'" />
      <carbon:warning class="size-5 shrink-0 stroke-current" v-else-if="toast.type === 'error'" />
      <carbon:warning class="size-5 shrink-0 stroke-current" v-else-if="toast.type === 'warning'" />
      <div class="min-w-0">
        <h3 class="text-lg font-bold max-md:text-base" v-if="toast.title">{{ toast.title }}</h3>
        <div v-html="toast.message" class="max-md:text-sm [&>a]:underline"></div>
      </div>
      <div class="shrink-0">
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
        <button class="btn btn-circle btn-xs" @click="removeToast(toast.id)" v-else>
          <mdi:close />
        </button>
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
const { toasts, removeToast } = useToast();
</script>
