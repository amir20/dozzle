<template>
  <div class="flex gap-2">
    <div
      class="flex min-w-[0.98rem] items-start justify-end align-bottom hover:cursor-pointer"
      v-if="isSupported"
      :title="t('log_actions.copy_log')"
    >
      <span
        class="text-primary rounded-sm bg-slate-800/60 px-1.5 py-1 hover:bg-slate-700"
        @click.prevent="copyLogMessageToClipBoard()"
      >
        <carbon:copy-file />
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { LogEntry, JSONObject } from "@/models/LogEntry";

const { message } = defineProps<{
  message: () => string;
  logEntry: LogEntry<string | JSONObject>;
}>();

const { showToast } = useToast();
const { copy, isSupported, copied } = useClipboard();
const { t } = useI18n();

async function copyLogMessageToClipBoard() {
  await copy(message());

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
