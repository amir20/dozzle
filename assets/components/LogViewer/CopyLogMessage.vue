<template>
  <div class="flex min-w-[0.98rem] items-start justify-end hover:cursor-pointer" title="Copy Log">
    <span
      class="hidden rounded px-1.5 py-[0.15rem] hover:bg-slate-700 hover:text-secondary group-hover/entry:block"
      @click="copyLogMessageToClipBoard()"
    >
      <carbon:copy-file />
    </span>
  </div>
</template>

<script lang="ts" setup>
const { message } = defineProps<{
  message: string;
}>();

const { showToast } = useToast();

function copyLogMessageToClipBoard() {
  navigator.clipboard.writeText(message);

  showToast(
    {
      title: "Copied",
      message: "Log message copied to clipboard",
      type: "info",
    },
    { expire: 2000 },
  );
}
</script>
