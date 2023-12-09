<template>
  <div
    class="flex min-w-[0.98rem] items-start justify-end align-bottom hover:cursor-pointer"
    v-if="isSupported && message.trim() != ''"
    :title="t('copy_log.title')"
  >
    <span
      class="rounded bg-slate-800/60 px-1.5 py-1 text-primary hover:bg-slate-700"
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

async function copyLogMessageToClipBoard() {
  await copy(message);

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
}
</script>
