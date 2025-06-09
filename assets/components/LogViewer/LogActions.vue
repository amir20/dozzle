<template>
  <div
    class="dropdown dropdown-start dropdown-hover font-sans group-[.compact]:absolute group-[.compact]:-left-0.5"
    v-show="container"
  >
    <button tabindex="0" class="btn btn-square btn-ghost btn-xs -mr-1 -ml-3 opacity-0 group-hover/entry:opacity-100">
      <ion:ellipsis-vertical />
    </button>
    <ul
      tabindex="0"
      class="menu dropdown-content rounded-box bg-base-200 border-base-content/20 z-50 -mr-1 -ml-3 w-52 border p-1 text-sm shadow-sm"
    >
      <li>
        <a>
          <material-symbols:content-copy />
          Copy line
        </a>
        <router-link
          :to="{
            name: '/container/[id].time.[datetime]',
            params: { id: container.id, datetime: logEntry.date.toISOString() },
            query: { logId: logEntry.id },
          }"
        >
          <material-symbols:link />
          Copy permalink
        </router-link>
        <router-link
          :to="{
            name: '/container/[id].time.[datetime]',
            params: { id: container.id, datetime: logEntry.date.toISOString() },
            query: { logId: logEntry.id },
          }"
        >
          <material-symbols:eye-tracking />
          See log in context
        </router-link>
        <a @click="showDrawer(LogDetails, { entry: logEntry })">
          <material-symbols:code-blocks-rounded />
          Show details
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { LogEntry, JSONObject } from "@/models/LogEntry";
import LogDetails from "./LogDetails.vue";

const { logEntry, container } = defineProps<{
  logEntry: LogEntry<string | JSONObject>;
  container: Container;
}>();

const { showToast } = useToast();
const showDrawer = useDrawer();

const { copy, isSupported, copied } = useClipboard();
const { t } = useI18n();

// async function copyLogMessageToClipBoard() {
//   await copy(message());

//   if (copied.value) {
//     showToast(
//       {
//         title: t("toasts.copied.title"),
//         message: t("toasts.copied.message"),
//         type: "info",
//       },
//       { expire: 2000 },
//     );
//   }
// }
</script>
