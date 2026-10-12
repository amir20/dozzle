<template>
  <li>
    <details>
      <summary>
        <div class="flex w-4 items-center gap-0.5">
          <carbon:circle-solid class="size-2" :class="streamConfig.stderr ? 'text-red' : 'opacity-20'" />
          <carbon:circle-solid class="size-2" :class="streamConfig.stdout ? 'text-blue' : 'opacity-20'" />
        </div>
        {{ $t("toolbar.streams") }}
        <span class="value">{{ streamSummary }}</span>
      </summary>
      <ul class="menu">
        <li v-for="option in streamOptions" :key="option.label">
          <a
            @click="
              streamConfig.stdout = option.stdout;
              streamConfig.stderr = option.stderr;
            "
          >
            <mdi:check
              class="w-4"
              v-if="streamConfig.stdout == option.stdout && streamConfig.stderr == option.stderr"
            />
            <div v-else class="w-4"></div>
            {{ option.label }}
          </a>
        </li>
      </ul>
    </details>
  </li>
  <li>
    <details class="group/details">
      <summary>
        <mdi:gauge />
        {{ $t("toolbar.levels") }}
        <span class="value group-open/details:hidden">{{ levelSummary }}</span>
        <Toggle
          class="toggle-xs hidden group-open/details:inline-flex"
          v-model="toggleAllLevels"
          :title="$t('toolbar.toggle-all-levels')"
        />
      </summary>
      <ul class="menu">
        <li v-for="level in allLevels">
          <a class="capitalize" @click="levels.has(level) ? levels.delete(level) : levels.add(level)">
            <mdi:check class="w-4" v-if="levels.has(level)" />
            <div v-else class="w-4"></div>

            <div class="flex">
              <div class="badge" :data-level="level">{{ level }}</div>
            </div>
          </a>
        </li>
      </ul>
    </details>
  </li>
</template>

<script lang="ts" setup>
import { allLevels } from "@/composable/logs/logContext";

// The Streams and Levels rows, shared by the filter chip and by the actions menu
// wherever the chip has no room (a phone, a pinned column). Renders bare `li`s,
// so it goes inside a `ul.menu.log-menu`.
const { streamConfig, levels } = useLoggingContext();
const { streamSummary, levelSummary } = useLogFilterSummary();
const { t } = useI18n();

const streamOptions = computed(() => [
  { label: t("toolbar.show-all"), stdout: true, stderr: true },
  { label: t("toolbar.show", { std: "STDOUT" }), stdout: true, stderr: false },
  { label: t("toolbar.show", { std: "STDERR" }), stdout: false, stderr: true },
]);

const toggleAllLevels = computed({
  get: () => levels.value.size === allLevels.length,
  set: (value) => {
    if (value) {
      allLevels.forEach((level) => levels.value.add(level));
    } else {
      levels.value.clear();
    }
  },
});
</script>
