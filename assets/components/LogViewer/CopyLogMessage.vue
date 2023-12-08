<template>
  <div
    class="absolute -right-1 flex min-w-[0.98rem] items-start justify-end align-bottom hover:cursor-pointer"
    v-if="isSupported && message.trim() != ''"
    title="Copy Log"
  >
    <span
      class="duration-250 rounded bg-slate-800/60 px-1.5 py-1 text-primary opacity-0 transition-opacity delay-100 hover:bg-slate-700 group-hover/entry:opacity-100"
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
const { copy, isSupported, copied } = useClipboard();
const { t } = useI18n();

function copyLogMessageToClipBoard() {
  copy(message).then(() => {
    if (copied.value) {
      showToast(
        {
          title: t("toasts.copied.title"),
          message: t("toasts.copied.message"),
          type: "info",
        },
        { expire: 2000 },
      );
    }
  });
}
</script>
