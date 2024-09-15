<template>
  <LogList :messages="visibleMessages" :show-container-name="showContainerName" />
</template>

<script lang="ts" setup>
import { type JSONObject, LogEntry } from "@/models/LogEntry";

const props = defineProps<{
  messages: LogEntry<string | JSONObject>[];
  visibleKeys: Map<string[], boolean>;
  showContainerName: boolean;
}>();

const { messages, visibleKeys } = toRefs(props);

const { filteredPayload } = useVisibleFilter(visibleKeys);
const { debouncedSearchFilter } = useSearchFilter();
const { streamConfig } = useLoggingContext();

const visibleMessages = filteredPayload(messages);
const router = useRouter();

watchEffect(() => {
  const query = {} as Record<string, string>;
  if (debouncedSearchFilter.value !== "") {
    query.search = debouncedSearchFilter.value;
  }

  if (!streamConfig.value.stderr) {
    query.stderr = streamConfig.value.stderr.toString();
  }

  if (!streamConfig.value.stdout) {
    query.stdout = streamConfig.value.stdout.toString();
  }

  router.push({
    query,
    replace: true,
  });
});
</script>
<style scoped lang="postcss"></style>
