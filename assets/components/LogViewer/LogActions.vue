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
      @click="hideMenu"
    >
      <li>
        <a v-if="isSupported" @click="copyLogMessage()">
          <material-symbols:content-copy />
          {{ $t("action.copy-log") }}
        </a>
        <a v-if="isSupported" @click="copyPermalink()">
          <material-symbols:link />
          {{ $t("action.copy-link") }}
        </a>
        <router-link
          v-if="isSearching"
          @click="resetSearch()"
          :to="{
            name: '/container/[id].time.[datetime]',
            params: { id: container.id, datetime: logEntry.date.toISOString() },
            query: { logId: logEntry.id },
          }"
        >
          <material-symbols:eye-tracking />
          {{ $t("action.see-in-context") }}
        </router-link>
        <a @click="showDrawer(LogDetails, { entry: logEntry })" v-if="logEntry instanceof ComplexLogEntry">
          <material-symbols:code-blocks-rounded />
          {{ $t("action.show-details") }}
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { LogEntry, SimpleLogEntry, ComplexLogEntry, JSONObject } from "@/models/LogEntry";
import LogDetails from "./LogDetails.vue";

const { logEntry, container } = defineProps<{
  logEntry: LogEntry<string | JSONObject>;
  container: Container;
}>();

const { showToast } = useToast();
const showDrawer = useDrawer();
const router = useRouter();
const { isSearching, resetSearch } = useSearchFilter();

const { copy, isSupported, copied } = useClipboard();
const { t } = useI18n();

async function copyLogMessage() {
  if (logEntry instanceof ComplexLogEntry) {
    await copy(logEntry.rawMessage);
  } else if (logEntry instanceof SimpleLogEntry) {
    await copy(logEntry.message);
  }

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

async function copyPermalink() {
  const url = router.resolve({
    name: "/container/[id].time.[datetime]",
    params: { id: container.id, datetime: logEntry.date.toISOString() },
    query: { logId: logEntry.id },
  }).href;

  const resolved = new URL(url, window.location.origin);

  await copy(resolved.href);

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

function hideMenu(e: MouseEvent) {
  if (e.target instanceof HTMLAnchorElement) {
    setTimeout(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    }, 50);
  }
}
</script>
