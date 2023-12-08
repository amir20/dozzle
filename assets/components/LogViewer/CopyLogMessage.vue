<template>
  <div
    class="absolute -right-1 flex min-w-[0.98rem] items-start justify-end align-bottom hover:cursor-pointer"
    v-if="message.trim() != ''"
    title="Copy Log"
  >
    <span
      class="rounded bg-slate-800/60 px-1.5 py-1 text-primary opacity-0 transition-opacity delay-500 duration-1000 hover:bg-slate-700 group-hover/entry:opacity-100"
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
