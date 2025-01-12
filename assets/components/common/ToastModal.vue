<template>
  <div class="toast toast-end whitespace-normal max-md:m-0 max-md:w-full">
    <div
      class="alert max-w-xl max-md:rounded-none"
      v-for="toast in toasts"
      :key="toast.id"
      :class="{
        'alert-error': toast.type === 'error',
        'alert-info': toast.type === 'info',
        'alert-warning': toast.type === 'warning',
      }"
    >
      <carbon:information class="size-6 shrink-0 stroke-current" v-if="toast.type === 'info'" />
      <carbon:warning class="size-6 shrink-0 stroke-current" v-else-if="toast.type === 'error'" />
      <carbon:warning class="size-6 shrink-0 stroke-current" v-else-if="toast.type === 'warning'" />
      <div>
        <h3 class="text-lg font-bold" v-if="toast.title">{{ toast.title }}</h3>
        <div v-html="toast.message" class="[&>a]:underline"></div>
      </div>
      <div>
        <button class="btn btn-circle btn-xs" @click="removeToast(toast.id)"><mdi:close /></button>
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
const { toasts, removeToast } = useToast();
</script>
