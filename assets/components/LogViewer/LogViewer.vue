<template>
  <LogList :messages="visibleMessages" />
</template>

<script lang="ts" setup>
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
  visibleKeys: Map<string[], boolean>;
}>();

const { messages, visibleKeys } = toRefs(props);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { debouncedSearchFilter, isSearching } = useSearchFilter();
const { streamConfig } = useLoggingContext();

const visibleMessages = filteredPayload(messages);
const router = useRouter();

// watchArray([debouncedSearchFilter, streamConfig], () => {
//   console.log("Updating route...");
//   const query = router.currentRoute.value.query;
//   const hash = router.currentRoute.value.hash;
//   if (isSearching.value) {
//     console.log("Searching...");
//     query.search = debouncedSearchFilter.value;
//   } else {
//     delete query.search;
//   }

//   if (!streamConfig.value.stderr) {
//     query.stderr = streamConfig.value.stderr.toString();
//   } else {
//     delete query.stderr;
//   }

//   if (!streamConfig.value.stdout) {
//     query.stdout = streamConfig.value.stdout.toString();
//   } else {
//     delete query.stdout;
//   }

//   console.log("Updating route...", query);
//   router.push({
//     query,
//     hash,
//     replace: true,
//   });
// });
//

// TODO
</script>
<style scoped></style>
