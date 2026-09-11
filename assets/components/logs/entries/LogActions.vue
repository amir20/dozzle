<template>
  <Popover
    v-if="container"
    hover
    placement="right-start"
    class="absolute -left-2 z-10 md:-left-8"
    panel-class="rounded-box bg-base-200 border-base-content/20 w-52 border p-1 font-sans text-sm shadow-sm"
  >
    <template #trigger>
      <router-link
        v-if="isFiltered"
        @click="resetSearch()"
        class="btn btn-square btn-xs border-base-content/20 bg-base-100 pointer-events-auto! opacity-0 shadow-sm group-hover/entry:opacity-90"
        :to="momentRoute"
      >
        <material-symbols:eye-tracking />
      </router-link>
      <button
        type="button"
        class="btn btn-square btn-xs border-base-content/20 bg-base-100 border opacity-0 shadow-sm group-hover/entry:opacity-90"
        v-else
      >
        <ion:ellipsis-vertical />
      </button>
    </template>
    <ul class="menu w-full p-0">
      <li v-if="isFiltered">
        <router-link @click="resetSearch()" :to="momentRoute">
          <material-symbols:eye-tracking />
          {{ $t("action.see-in-context") }}
        </router-link>
      </li>
      <li>
        <a @click="copyLogMessage()">
          <material-symbols:content-copy />
          {{ $t("action.copy-log") }}
        </a>
      </li>
      <li>
        <a @click="copyPermalink()">
          <material-symbols:link />
          {{ $t("action.copy-link") }}
        </a>
      </li>

      <li v-if="logEntry instanceof ComplexLogEntry">
        <a @click="showDrawer(LogDetails, { entry: logEntry })">
          <material-symbols:code-blocks-rounded />
          {{ $t("action.show-details") }}
        </a>
      </li>
      <li>
        <a @click="createAlert()">
          <mdi:bell />
          {{ $t("action.create-alert") }}
        </a>
      </li>
      <li v-if="hasCloudChat">
        <a @click="askAboutLine(toViewLogLine(logEntry))">
          <mdi:message-question-outline />
          {{ $t("action.ask-about-log") }}
        </a>
      </li>
    </ul>
  </Popover>
</template>

<script lang="ts" setup>
import stripAnsi from "strip-ansi";
import { Container } from "@/models/Container";
import { LogEntry, SimpleLogEntry, ComplexLogEntry, GroupedLogEntry, JSONObject } from "@/models/LogEntry";
import { toViewLogLine } from "@/composable/logs/viewContext";
import LogDetails from "./LogDetails.vue";
import AlertForm from "@/components/notifications/AlertForm.vue";

const { logEntry, container } = defineProps<{
  logEntry: LogEntry<string | JSONObject>;
  container: Container;
}>();

const showDrawer = useDrawer();
// Cloud owns the answer, so the row is absent rather than dead when nobody has
// linked an account.
const { linked: hasCloudChat } = useCloudSurface();
const { askAboutLine } = useCloudChat();
const { hrefFor } = useLogJump();
const moment = () => ({ containerId: container.id, date: logEntry.date, logId: logEntry.id });
const momentRoute = computed(() => logMomentRoute(moment()));
const { isSearching, resetSearch } = useSearchFilter();
const { levels } = useLoggingContext();

// Show "see in context" whenever the stream is narrowed, either by a text search
// or by a log-level filter, so the entry can be inspected in the full log stream.
const isFiltered = computed(() => isSearching.value || allLevels.some((level) => !levels.value.has(level)));

const { copy } = useCopy();

async function copyLogMessage() {
  if (logEntry instanceof ComplexLogEntry) {
    await copy(stripAnsi(logEntry.rawMessage));
  } else if (logEntry instanceof SimpleLogEntry) {
    await copy(stripAnsi(logEntry.rawMessage));
  } else if (logEntry instanceof GroupedLogEntry) {
    await copy(stripAnsi(logEntry.message.join("\n")));
  }
}

async function copyPermalink() {
  await copy(hrefFor(moment()));
}

function createAlert() {
  const containerExpr = `name contains "${container.name}"`;
  let logExpr = "";
  if (logEntry.level && logEntry.level !== "unknown") {
    logExpr = `level == "${logEntry.level}"`;
  }

  const nameParts = [container.name];
  if (logEntry.level && logEntry.level !== "unknown") {
    nameParts.push(logEntry.level);
  }
  const name = nameParts.join(" ");

  showDrawer(AlertForm, { prefill: { name, containerExpression: containerExpr, logExpression: logExpr } }, "lg");
}
</script>
