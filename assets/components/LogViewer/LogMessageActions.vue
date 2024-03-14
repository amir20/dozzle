<template>
  <div class="flex gap-2">
    <div
      class="flex min-w-[0.98rem] items-start justify-end align-bottom hover:cursor-pointer"
      v-if="isSupported"
      :title="t('log_actions.copy_log')"
    >
      <span
        class="rounded bg-slate-800/60 px-1.5 py-1 text-primary hover:bg-slate-700"
        @click="copyLogMessageToClipBoard()"
      >
        <carbon:copy-file />
      </span>
    </div>
    <div
      class="flex min-w-[0.98rem] items-start justify-end align-bottom hover:cursor-pointer"
      :title="t('log_actions.jump_to_context')"
      v-if="isSearching()"
    >
      <a
        class="rounded bg-slate-800/60 px-1.5 py-1 text-primary hover:bg-slate-700"
        @click="handleJumpLineSelected($event, logEntry)"
        :href="`#${logEntry.id}`"
      >
        <carbon:search-locate />
      </a>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { LogEntry, JSONObject } from "@/models/LogEntry";

const { message, logEntry } = defineProps<{
  message: () => string;
  logEntry: LogEntry<string | JSONObject>;
}>();

const { showToast } = useToast();
const { copy, isSupported, copied } = useClipboard();
const { t } = useI18n();

const { isSearching } = useSearchFilter();
const { handleJumpLineSelected } = useLogSearchContext();

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
